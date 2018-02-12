package ipallocator

import (
	"context"
	"fmt"
	"math"
	"net"
	"sync"

	"github.com/Scalingo/go-etcd-lock/lock"
	"github.com/Scalingo/go-internal-tools/logger"
	"github.com/Scalingo/sand/config"
	"github.com/Scalingo/sand/store"
	"github.com/pkg/errors"
	"github.com/willf/bitset"
)

const (
	lockDuration          = 20
	IPAllocatorPrefix     = "/ipalloc"
	IPAllocatorLockPrefix = "/ipalloc-lock"
	DefaultAddressRange   = "10.0.0.0/24"
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

type IPAllocator interface {
	AllocateIP(ctx context.Context, id string, opts AllocateIPOpts) (string, error)
	ReleaseIP(ctx context.Context, id string, address string) error
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

func (a *allocator) lockStorageKey(id string) string {
	return fmt.Sprintf("%s/%s", IPAllocatorLockPrefix, id)
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
		// Modifies allocation.BitSet
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
	log.Info("allocating IP")

	_, addressNet, err := net.ParseCIDR(opts.AddressRange)
	if err != nil {
		return alloc, errors.Wrapf(err, "invalid iprange %v", opts.AddressRange)
	}

	mask, bits := addressNet.Mask.Size()
	// 0.0.0.0/24 -> mask = 24, bits = 32
	// 2^8 -> 256 addresses
	addressCount := uint(math.Pow(2.0, float64(bits-mask)))

	err = a.store.Get(ctx, alloc.storageKey(), false, &alloc)
	if err == store.ErrNotFound {
		alloc.AddressRange = opts.AddressRange
		alloc.AddressCount = addressCount
		// Network and Broadcast addresses are reserved
		alloc.BitSet = bitset.New(alloc.AddressCount).Set(0).Set(addressCount - 1)
		return alloc, nil
	}
	if err != nil {
		return alloc, errors.Wrapf(err, "fail to get allocation from storage")
	}

	if alloc.AddressRange != opts.AddressRange {
		return alloc, errors.Errorf("invalid IP range %v, allocation already exists with range %v", opts.AddressRange, alloc.AddressRange)
	}

	return alloc, nil
}

func (a allocation) allocatePredefinedIP(ctx context.Context, address string) error {
	log := logger.Get(ctx)

	addrIP, addressIpnet, err := net.ParseCIDR(address)
	if err != nil {
		return errors.Wrapf(err, "fail to parse predefined address ip range '%v'", address)
	}

	if addressIpnet.Network() != a.AddressRange {
		return errors.Wrapf(err, "predefined address is not in the same ip range: %v != %v", addressIpnet.Network(), a.AddressRange)
	}

	ordinal := ordinalFromIP4(addrIP, addressIpnet.Mask)
	if a.BitSet.Test(ordinal) {
		return errors.Wrapf(err, "ip is already allocated")
	}
	a.BitSet.Set(ordinal)

	log.WithField("ip", addrIP).WithField("ip-range", a.AddressRange).Info("allocated predefined IP")

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
	addIntToIP(ip, uint64(i))

	log.WithField("ip", ip).WithField("ip-range", a.AddressRange).Info("allocated IP")

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

	ip, _, err := net.ParseCIDR(ipcidr)
	if err != nil {
		return errors.Wrapf(err, "fail to parse IP CIDR %v", ipcidr)
	}

	_, network, err := net.ParseCIDR(alloc.AddressRange)
	if err != nil {
		return errors.Wrapf(err, "fail to parse iprange of allocation %v", alloc.AddressRange)
	}

	log = log.WithField("ip", ip).WithField("ip-range", alloc.AddressRange)
	i := ordinalFromIP4(ip, network.Mask)
	log.WithField("ordinal", i).Debug("IP ordinal")
	alloc.BitSet.Clear(i)
	log.Info("release IP")

	err = a.store.Set(ctx, alloc.storageKey(), &alloc)
	if err != nil {
		return errors.Wrapf(err, "fail to store ip range in store")
	}
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

// Adds the ordinal IP to the current array
// 192.168.0.0 + 53 => 192.168.0.53
func addIntToIP(array []byte, ordinal uint64) {
	for i := len(array) - 1; i >= 0; i-- {
		array[i] |= (byte)(ordinal & 0xff)
		ordinal >>= 8
	}
}
