package ipallocator

import (
	"context"
	"fmt"
	"math"
	"net"

	"github.com/Scalingo/go-internal-tools/logger"
	"github.com/Scalingo/sand/config"
	"github.com/Scalingo/sand/store"
	"github.com/pkg/errors"
	"github.com/willf/bitset"
)

const (
	IPAllocatorPrefix = "/ipalloc"
	DefaultIPRange    = "10.0.0.0/24"
)

type AllocatorOpt func(*allocator)

func WithIPRange(r string) AllocatorOpt {
	return func(a *allocator) {
		a.ipRange = r
	}
}

type AllocateIPOpts struct {
	// If set, will try to allocate this precise IP, error if already taken
	Address string
}

type IPAllocator interface {
	AllocateIP(ctx context.Context, opts AllocateIPOpts) (net.IP, uint, error)
	ReleaseIP(ctx context.Context, ip net.IP) error
}

type allocator struct {
	config        *config.Config
	store         store.Store
	id            string
	ipRange       string
	ipNet         *net.IPNet
	addressAmount uint
	mask          uint
	err           error
}

func New(config *config.Config, store store.Store, id string, opts ...AllocatorOpt) IPAllocator {
	a := &allocator{config: config, store: store, id: id}
	for _, opt := range opts {
		opt(a)
	}
	if a.ipRange == "" {
		a.ipRange = DefaultIPRange
	}

	_, ipnet, err := net.ParseCIDR(a.ipRange)
	if err != nil {
		a.err = errors.Wrapf(err, "fail to parse ip range '%v'", a.ipRange)
		return a
	}
	a.ipNet = ipnet

	mask, size := a.ipNet.Mask.Size()
	if mask == 0 {
		a.err = errors.Errorf("invalid mask in ip range '%v'", a.ipRange)
		return a
	}

	a.mask = uint(mask)

	a.addressAmount = uint(math.Pow(2.0, float64(size-mask)))
	return a
}

func (a *allocator) StorageKey() string {
	return fmt.Sprintf("%s/%s", IPAllocatorPrefix, a.id)
}

func (a *allocator) AllocateIP(ctx context.Context, opts AllocateIPOpts) (net.IP, uint, error) {
	if a.err != nil {
		return nil, 0, errors.Wrapf(a.err, "invalid allocator")
	}

	if opts.Address != "" {
		return a.allocatePredefinedIP(ctx, opts.Address)
	}

	return a.allocateNextAvailableIP(ctx)
}

func (a *allocator) allocatePredefinedIP(ctx context.Context, address string) (net.IP, uint, error) {
	log := logger.Get(ctx)

	addrIP, addressIpnet, err := net.ParseCIDR(address)
	if err != nil {
		return nil, 0, errors.Wrapf(err, "fail to parse predefined address ip range '%v'", address)
	}

	if addressIpnet.Network() != a.ipNet.Network() {
		return nil, 0, errors.Wrapf(err, "predefined address is not in the same ip range: %v != %v", addressIpnet.Network(), a.ipNet.Network())
	}

	r, err := a.getBitset(ctx)
	if err != nil {
		return nil, 0, errors.Wrapf(err, "fail to get bitset")
	}

	ordinal := ordinalFromIP4(addrIP, addressIpnet.Mask)
	if r.Test(ordinal) {
		return nil, 0, errors.Wrapf(err, "ip is already allocated")
	}

	r.Set(ordinal)
	log.WithField("ip", addrIP).WithField("ip-range", a.ipRange).Info("allocate predefined IP")

	err = a.store.Set(ctx, a.StorageKey(), &r)
	if err != nil {
		return addrIP, 0, errors.Wrapf(err, "fail to store ip range in store")
	}

	return addrIP, a.mask, nil
}

func (a *allocator) getBitset(ctx context.Context) (*bitset.BitSet, error) {
	var r *bitset.BitSet
	err := a.store.Get(ctx, a.StorageKey(), false, &r)
	if err == store.ErrNotFound {
		r = bitset.New(a.addressAmount)
		// Network and Broadcast addresses
		r.Set(0).Set(r.Len() - 1)
	} else if err != nil {
		return nil, errors.Wrapf(err, "fail to get ip range from store")
	}
	if r.Len() != a.addressAmount {
		return nil, errors.Errorf("range stored does not fit required range for allocator %v != %v", r.Len(), a.addressAmount)
	}
	return r, nil
}

func (a *allocator) allocateNextAvailableIP(ctx context.Context) (net.IP, uint, error) {
	log := logger.Get(ctx)
	r, err := a.getBitset(ctx)
	if err != nil {
		return nil, 0, errors.Wrapf(err, "fail to get bitset")
	}

	i := uint(0)
	for ; i < a.addressAmount; i++ {
		if !r.Test(i) {
			r.Set(i)
			break
		}
	}

	ip := a.ipNet.IP
	addIntToIP(ip, uint64(i))

	log.WithField("ip", ip).WithField("ip-range", a.ipRange).Info("allocate IP")

	err = a.store.Set(ctx, a.StorageKey(), &r)
	if err != nil {
		return ip, 0, errors.Wrapf(err, "fail to store ip range in store")
	}

	return ip, a.mask, nil
}

func (a *allocator) ReleaseIP(ctx context.Context, ip net.IP) error {
	log := logger.Get(ctx)

	var r *bitset.BitSet
	err := a.store.Get(ctx, a.StorageKey(), false, &r)
	if err != nil {
		return errors.Wrapf(err, "fail to get ip range from store")
	}

	_, network, err := net.ParseCIDR(a.ipRange)
	if err != nil {
		return errors.Wrapf(err, "fail to parse iprange of allocator %v", a.ipRange)
	}

	log = log.WithField("ip", ip).WithField("ip-range", a.ipRange)
	i := ordinalFromIP4(ip, network.Mask)
	log.WithField("ordinal", i).Debug("IP ordinal")
	r.Clear(i)

	log.Info("release IP")

	err = a.store.Set(ctx, a.StorageKey(), &r)
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
