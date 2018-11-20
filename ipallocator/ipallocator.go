package ipallocator

import (
	"context"
	"fmt"
	"math"
	"net"
	"strings"
	"sync"

	"github.com/Scalingo/go-etcd-lock/lock"
	"github.com/Scalingo/go-internal-tools/logger"
	"github.com/Scalingo/sand/api/types"
	"github.com/Scalingo/sand/config"
	"github.com/Scalingo/sand/netutils"
	"github.com/Scalingo/sand/store"
	"github.com/pkg/errors"
	"github.com/willf/bitset"
)

const (
	lockDuration          = 20
	IPAllocatorPrefix     = "/ipalloc"
	IPAllocatorLockPrefix = "/ipalloc-lock"
)

type allocation struct {
	ID           string         `json:"id"`
	AddressRange string         `json:"address_range"`
	AddressCount uint           `json:"address_count"`
	BitSet       *bitset.BitSet `json:"bit_set"`
}

func (a allocation) storageKey() string {
	return fmt.Sprintf("%s/%s", IPAllocatorPrefix, a.ID)
}

type AllocateIPOpts struct {
	AddressRange string
	// If set, will try to allocate this precise IP, error if already taken
	Address string
}

type RangeAddresser interface {
	GetAddressRange() string
}

type IPAllocator interface {
	InitializePool(ctx context.Context, id string, addressRange string) (RangeAddresser, error)
	AllocateIP(ctx context.Context, id string, opts AllocateIPOpts) (string, error)
	ReleaseIP(ctx context.Context, id string, address string) error
	ReleasePool(ctx context.Context, id string) error
}

type allocator struct {
	config      *config.Config
	store       store.Store
	locker      lock.Locker
	m           *sync.Mutex
	allocations map[string]allocation
}

func New(config *config.Config, store store.Store, locker lock.Locker) *allocator {
	return &allocator{
		config: config, store: store,
		locker:      locker,
		m:           &sync.Mutex{},
		allocations: make(map[string]allocation),
	}
}

func (a *allocation) GetAddressRange() string {
	return a.AddressRange
}

func (a *allocator) lockStorageKey(id string) string {
	return fmt.Sprintf("%s/%s", IPAllocatorLockPrefix, id)
}

func (a *allocator) InitializePool(ctx context.Context, id string, addressRange string) (RangeAddresser, error) {
	if addressRange == "" {
		addressRange = types.DefaultIPRange
	}
	allocation, err := a.findOrCreateAllocation(ctx, id, AllocateIPOpts{AddressRange: addressRange})
	if err != nil {
		return nil, errors.Wrapf(err, "fail to create allocation")
	}

	err = a.store.Set(ctx, allocation.storageKey(), &allocation)
	if err != nil {
		return nil, errors.Wrapf(err, "fail to save updated allocation %v", allocation)
	}

	return &allocation, nil
}

func (a *allocator) AllocateIP(ctx context.Context, id string, opts AllocateIPOpts) (allocatedAddress string, err error) {
	lock, err := a.locker.WaitAcquire(a.lockStorageKey(id), lockDuration)
	if err != nil {
		return "", errors.Wrapf(err, "fail to lock IP allocation")
	}
	defer func() {
		derr := lock.Release()
		if derr != nil {
			err = errors.Wrapf(derr, "fail to release lock (err is %v)", err)
		}
	}()

	allocation, err := a.findOrCreateAllocation(ctx, id, opts)
	if err != nil {
		return "", errors.Wrapf(err, "fail to find or create allocation")
	}

	allocatedAddress = opts.Address
	if allocatedAddress != "" {
		if !strings.Contains(allocatedAddress, "/") {
			_, addressNet, err := net.ParseCIDR(allocation.AddressRange)
			if err != nil {
				return "", errors.Wrapf(err, "invalid iprange %v", allocation.AddressRange)
			}
			ones, _ := addressNet.Mask.Size()
			allocatedAddress = fmt.Sprintf("%s/%d", allocatedAddress, ones)
		}
		err = allocation.allocatePredefinedIP(ctx, allocatedAddress)
	} else {
		allocatedAddress, err = allocation.allocateNextAvailableIP(ctx)
	}
	if err != nil {
		return "", errors.Wrapf(err, "fail to allocate IP")
	}

	err = a.store.Set(ctx, allocation.storageKey(), &allocation)
	if err != nil {
		return "", errors.Wrapf(err, "fail to save updated allocation %v", allocation)
	}

	return allocatedAddress, err
}

func (a *allocator) findOrCreateAllocation(ctx context.Context, id string, opts AllocateIPOpts) (allocation, error) {
	var (
		err   error
		alloc = allocation{
			ID: id,
		}
	)

	log := logger.Get(ctx).WithField("allocation_id", id)
	log.Info("find or create allocation")

	err = a.store.Get(ctx, alloc.storageKey(), false, &alloc)
	if err == store.ErrNotFound {
		_, addressNet, err := net.ParseCIDR(opts.AddressRange)
		if err != nil {
			return alloc, errors.Wrapf(err, "invalid iprange %v", opts.AddressRange)
		}
		mask, bits := addressNet.Mask.Size()
		// 0.0.0.0/24 -> mask = 24, bits = 32
		// 2^8 -> 256 addresses
		addressCount := uint(math.Pow(2.0, float64(bits-mask)))

		alloc.AddressRange = opts.AddressRange
		alloc.AddressCount = addressCount
		// Network and Broadcast addresses are reserved
		alloc.BitSet = bitset.New(alloc.AddressCount).Set(0).Set(addressCount - 1)
	} else if err != nil {
		return alloc, errors.Wrapf(err, "fail to get allocation from storage")
	}

	return alloc, nil
}

func (a allocation) allocatePredefinedIP(ctx context.Context, address string) error {
	log := logger.Get(ctx)

	addrIP, addressIpnet, err := net.ParseCIDR(address)
	if err != nil {
		return errors.Wrapf(err, "fail to parse predefined address ip range '%v'", address)
	}
	log.WithField("ip", addrIP).WithField("ip-range", a.AddressRange).Info("allocation of predefined IP")

	if addressIpnet.String() != a.AddressRange {
		return errors.Errorf("predefined address is not in the same ip range: %v != %v", addressIpnet.Network(), a.AddressRange)
	}

	ordinal := ordinalFromIP4(addrIP, addressIpnet.Mask)
	if a.BitSet.Test(ordinal) {
		return errors.New("ip is already allocated")
	}
	a.BitSet.Set(ordinal)

	log.WithField("ip", addrIP).WithField("ip-range", a.AddressRange).Infof("allocated predefined IP (bitset %d)", ordinal)
	return nil
}

func (a allocation) allocateNextAvailableIP(ctx context.Context) (string, error) {
	log := logger.Get(ctx)

	i := uint(0)
	for ; i < a.AddressCount; i++ {
		if !a.BitSet.Test(i) {
			a.BitSet.Set(i)
			break
		}
	}

	ip, ipnet, err := net.ParseCIDR(a.AddressRange)
	if err != nil {
		return "", errors.Wrapf(err, "fail to parse allocation address range %v", a.AddressRange)
	}
	ip = netutils.AddIntToIP(ip, uint64(i))

	log.WithField("ip", ip).WithField("ip-range", a.AddressRange).Infof("allocated IP (bitset %d)", i)

	ones, _ := ipnet.Mask.Size()
	return fmt.Sprintf("%s/%d", ip.String(), ones), nil
}

func (a *allocator) ReleaseIP(ctx context.Context, id string, ipcidr string) (err error) {
	log := logger.Get(ctx)

	lock, err := a.locker.WaitAcquire(a.lockStorageKey(id), lockDuration)
	if err != nil {
		return errors.Wrapf(err, "fail to lock IP release")
	}
	defer func() {
		derr := lock.Release()
		if derr != nil {
			err = errors.Wrapf(err, "fail to release lock when releasing IP (err is %v)", err)
		}
	}()

	alloc := allocation{ID: id}
	err = a.store.Get(ctx, alloc.storageKey(), false, &alloc)
	if err != nil {
		return errors.Wrapf(err, "fail to get ip range from store")
	}

	_, network, err := net.ParseCIDR(alloc.AddressRange)
	if err != nil {
		return errors.Wrapf(err, "fail to parse iprange of allocation %v", alloc.AddressRange)
	}
	if !strings.Contains(ipcidr, "/") {
		ones, _ := network.Mask.Size()
		ipcidr = fmt.Sprintf("%s/%d", ipcidr, ones)
	}

	ip, _, err := net.ParseCIDR(ipcidr)
	if err != nil {
		return errors.Wrapf(err, "fail to parse IP CIDR %v", ipcidr)
	}

	log = log.WithField("ip", ip).WithField("ip-range", alloc.AddressRange)
	i := ordinalFromIP4(ip, network.Mask)
	log.WithField("ordinal", i).Debug("IP ordinal")
	alloc.BitSet.Clear(i)
	log.Infof("release IP (bitset %d)", i)

	err = a.store.Set(ctx, alloc.storageKey(), &alloc)
	if err != nil {
		return errors.Wrapf(err, "fail to store ip range in store")
	}
	return nil
}

func (a *allocator) ReleasePool(ctx context.Context, id string) (err error) {
	log := logger.Get(ctx).WithField("allocation_id", id)
	lock, err := a.locker.WaitAcquire(a.lockStorageKey(id), lockDuration)
	if err != nil {
		return errors.Wrapf(err, "fail to lock IP release pool")
	}
	defer func() {
		derr := lock.Release()
		if derr != nil {
			err = errors.Wrapf(err, "fail to release lock when releasing pool (err is %v)", err)
		}
	}()

	alloc := allocation{ID: id}
	log.Infof("Releasing allocation")
	err = a.store.Get(ctx, alloc.storageKey(), false, &alloc)
	if err != nil {
		return errors.Wrapf(err, "fail to get ip range from store")
	}
	if err == store.ErrNotFound {
		log.Infof("allocation not found %v", alloc)
		return nil
	}

	err = a.store.Delete(ctx, alloc.storageKey())
	if err != nil {
		return errors.Wrapf(err, "fail to delete ip range reference")
	}

	log.Info("Allocation deleted")

	return nil
}

func ordinalFromIP4(ip net.IP, mask net.IPMask) uint {
	var (
		ordinal uint = 0
		rank    uint = 1
	)
	// inv mask
	for i := range mask {
		mask[i] = mask[i] ^ 0xff
	}
	ip = ip.Mask(mask)
	for i := len(ip) - 1; i >= 0; i-- {
		ordinal += uint(ip[i]) * rank
		rank++
	}
	return ordinal
}
