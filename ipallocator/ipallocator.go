package ipallocator

import (
	"context"
	"fmt"
	"math"
	"net"

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

type IPAllocator interface {
	AllocateIP(ctx context.Context) (net.IP, uint, error)
}

type allocator struct {
	config  *config.Config
	store   store.Store
	id      string
	ipRange string
}

func New(config *config.Config, store store.Store, id string, opts ...AllocatorOpt) IPAllocator {
	a := &allocator{config: config, store: store, id: id}
	for _, opt := range opts {
		opt(a)
	}
	return a
}

func (a *allocator) StorageKey() string {
	return fmt.Sprintf("%s/%s", IPAllocatorPrefix, a.id)
}

func (a *allocator) AllocateIP(ctx context.Context) (net.IP, uint, error) {
	if a.ipRange == "" {
		a.ipRange = DefaultIPRange
	}

	_, ipnet, err := net.ParseCIDR(a.ipRange)
	if err != nil {
		return nil, 0, errors.Wrapf(err, "fail to parse ip range '%v'", a.ipRange)
	}
	mask, size := ipnet.Mask.Size()
	if mask == 0 {
		return nil, 0, errors.Errorf("invalid mask in ip range '%v'", a.ipRange)
	}
	addressAmount := uint(math.Pow(2.0, float64(size-mask)))

	var r *bitset.BitSet
	err = a.store.Get(ctx, a.StorageKey(), false, &r)
	if err == store.ErrNotFound {
		r = bitset.New(addressAmount)
		// Network and Broadcast addresses
		r.Set(0).Set(r.Len() - 1)
	} else if err != nil {
		return nil, 0, errors.Wrapf(err, "fail to get ip range from store")
	}
	if r.Len() != addressAmount {
		return nil, 0, errors.Errorf("range stored does not fit required range for allocator %v != %v", r.Len(), addressAmount)
	}

	i := uint(0)
	for ; i < addressAmount; i++ {
		if !r.Test(i) {
			r.Set(i)
			break
		}
	}

	ip := ipnet.IP
	addIntToIP(ip, uint64(i))

	err = a.store.Set(ctx, a.StorageKey(), &r)
	if err != nil {
		return ip, 0, errors.Wrapf(err, "fail to store ip range in store")
	}

	return ip, uint(mask), nil
}

// Adds the ordinal IP to the current array
// 192.168.0.0 + 53 => 192.168.0.53
func addIntToIP(array []byte, ordinal uint64) {
	for i := len(array) - 1; i >= 0; i-- {
		array[i] |= (byte)(ordinal & 0xff)
		ordinal >>= 8
	}
}
