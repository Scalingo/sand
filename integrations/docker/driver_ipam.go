package docker

import (
	"context"

	"github.com/Scalingo/go-plugins-helpers/ipam"
	"github.com/Scalingo/go-utils/logger"
	"github.com/Scalingo/sand/api/types"
	"github.com/Scalingo/sand/ipallocator"
	"github.com/Scalingo/sand/network"
	"github.com/pkg/errors"
)

type dockerIPAMPlugin struct {
	allocator         ipallocator.IPAllocator
	networkRepository network.Repository
}

func (p *dockerIPAMPlugin) GetCapabilities(context.Context) (*ipam.CapabilitiesResponse, error) {
	return &ipam.CapabilitiesResponse{
		RequiresMACAddress: false,
	}, nil
}

func (p *dockerIPAMPlugin) GetDefaultAddressSpaces(context.Context) (*ipam.AddressSpacesResponse, error) {
	return &ipam.AddressSpacesResponse{
		LocalDefaultAddressSpace:  types.DefaultIPRange,
		GlobalDefaultAddressSpace: types.DefaultGateway,
	}, nil
}

func (p *dockerIPAMPlugin) RequestPool(ctx context.Context, req *ipam.RequestPoolRequest) (*ipam.RequestPoolResponse, error) {
	log := logger.Get(ctx)

	id := req.Options["sand-id"]
	if id == "" {
		return nil, errors.New("IPAM option sand-id is mandatory")
	}

	_, ok, err := p.networkRepository.Exists(ctx, id)
	if err != nil {
		return nil, errors.Wrapf(err, "fail to get network %v", id)
	}
	if !ok {
		return nil, errors.Errorf("SAND network %v does not exist", id)
	}

	allocation, err := p.allocator.InitializePool(ctx, id, req.Pool)
	if err != nil {
		return nil, errors.Wrapf(err, "fail to initialize IP pool")
	}
	log.Info("pool initialized")
	res := ipam.RequestPoolResponse{
		PoolID: id,
		Pool:   allocation.GetAddressRange(),
		Data:   map[string]string{},
	}
	return &res, nil
}

func (p *dockerIPAMPlugin) ReleasePool(ctx context.Context, req *ipam.ReleasePoolRequest) error {
	// Always deleted through SAND API
	return nil
}

func (p *dockerIPAMPlugin) RequestAddress(ctx context.Context, req *ipam.RequestAddressRequest) (*ipam.RequestAddressResponse, error) {
	log := logger.Get(ctx)
	log = log.WithField("pool_id", req.PoolID)

	ip, err := p.allocator.AllocateIP(ctx, req.PoolID, ipallocator.AllocateIPOpts{
		Address: req.Address,
	})
	if err != nil {
		return nil, errors.Wrapf(err, "fail to request address in pool %v - %v", req.PoolID, req.Address)
	}
	log.Infof("obtained address: %v", ip)
	return &ipam.RequestAddressResponse{
		Address: ip,
	}, nil
}

func (p *dockerIPAMPlugin) ReleaseAddress(ctx context.Context, req *ipam.ReleaseAddressRequest) error {
	log := logger.Get(ctx)
	log = log.WithField("pool_id", req.PoolID)

	err := p.allocator.ReleaseIP(ctx, req.PoolID, req.Address)
	if err != nil {
		return errors.Wrapf(err, "fail to release address in pool %v - %v", req.PoolID, req.Address)
	}
	log.Infof("released address: %v", req.Address)
	return nil
}
