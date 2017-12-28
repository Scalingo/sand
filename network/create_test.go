package network

import (
	"context"
	"errors"
	"testing"

	"github.com/Scalingo/sand/api/types"
	"github.com/Scalingo/sand/config"
	"github.com/Scalingo/sand/network/netmanager"
	"github.com/Scalingo/sand/test/mocks/network/netmanagermock"
	"github.com/Scalingo/sand/test/mocks/storemock"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/require"
)

// func NewRepository(config *config.Config, store store.Store, listener overlay.NetworkEndpointListener) Repository {
// 	return &repository{
// 		config: config, store: store, listener: listener,
// 		managers: map[types.NetworkType]netmanager.NetManager{
// 			types.OverlayNetworkType: overlay.NewManager(config, listener),
// 		},
// 	}
// }

func TestRepository_Ensure(t *testing.T) {
	cases := []struct {
		Name             string
		Network          func() types.Network
		Error            string
		ExpectNetManager func(*netmanagermock.MockNetManager, types.Network)
	}{
		{
			Name: "network with unknown type should return an error",
			Network: func() types.Network {
				return types.Network{Type: types.NetworkType("unknown")}
			},
			Error: "invalid",
		}, {
			Name: "overlay: should return error if manager Ensure fails",
			ExpectNetManager: func(m *netmanagermock.MockNetManager, n types.Network) {
				m.EXPECT().Ensure(gomock.Any(), n).Return(errors.New("fail to ensure network"))
			},
			Error: "fail to ensure",
		},
	}

	for _, c := range cases {
		t.Run(c.Name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			config, err := config.Build()
			require.NoError(t, err)

			omanager := netmanagermock.NewMockNetManager(ctrl)
			managers := map[types.NetworkType]netmanager.NetManager{
				types.OverlayNetworkType: omanager,
			}
			store := storemock.NewMockStore(ctrl)

			r := repository{
				config:   config,
				store:    store,
				managers: managers,
			}

			network := types.Network{ID: "1", Type: types.OverlayNetworkType}
			if c.Network != nil {
				network = c.Network()
			}

			if c.ExpectNetManager != nil {
				c.ExpectNetManager(omanager, network)
			}

			err = r.Ensure(context.Background(), network)
			if c.Error != "" {
				require.Error(t, err)
				require.Contains(t, err.Error(), c.Error)
				return
			}
			require.NoError(t, err)
		})
	}
}

// func (c *repository) Ensure(ctx context.Context, network types.Network) error {
// 	switch network.Type {
// 	case types.OverlayNetworkType:
// 		m := c.managers[network.Type]
// 		err := m.Ensure(ctx, network)
// 		if err != nil {
// 			return errors.Wrapf(err, "fail to ensure overlay network %s", network)
// 		}
// 		var endpoints []types.Endpoint
// 		err = c.store.Get(ctx, network.EndpointsStorageKey(""), true, &endpoints)
// 		if err != nil && err != store.ErrNotFound {
// 			return errors.Wrapf(err, "fail to get network endpoints")
// 		}

// 		if len(endpoints) > 0 {
// 			err = m.EnsureEndpointsNeigh(ctx, network, endpoints)
// 			if err != nil {
// 				return errors.Wrapf(err, "fail to ensure neighbors (ARP/FDB)")
// 			}
// 		}

// 		err = c.listener.Add(ctx, m, network)
// 		if err != nil {
// 			return errors.Wrapf(err, "fail to listen for new endpoints on network '%s'", network)
// 		}
// 	default:
// 		return errors.New("invalid network type")
// 	}

// 	// Ability to list all networks with node hostname as prefix
// 	err := c.store.Set(
// 		ctx,
// 		fmt.Sprintf("/nodes/%s/networks/%s", c.config.PublicHostname, network.ID),
// 		map[string]interface{}{"id": network.ID, "created_at": time.Now()},
// 	)
// 	if err != nil {
// 		return errors.Wrapf(err, "err to store nodes link to network %s", network)
// 	}

// 	// Ability to list nodes present in a network
// 	err = c.store.Set(
// 		ctx,
// 		fmt.Sprintf("/nodes-networks/%s/%s", network.ID, c.config.PublicHostname),
// 		map[string]interface{}{"id": network.ID, "created_at": time.Now()},
// 	)
// 	if err != nil {
// 		return errors.Wrapf(err, "err to store network %s link to hostname", network)
// 	}

// 	return nil
// }

// func (c *repository) Create(ctx context.Context, params params.CreateNetworkParams) (types.Network, error) {
// 	log := logger.Get(ctx).WithField("network_name", params.Name)

// 	if params.Type == "" {
// 		params.Type = types.OverlayNetworkType
// 	}

// 	network, err := c.new(ctx, params)
// 	if err != nil {
// 		return network, errors.Wrapf(err, "fail to initialize network %s", params.Name)
// 	}

// 	log = log.WithField("network_id", network.ID)
// 	ctx = logger.ToCtx(ctx, log)

// 	err = c.Ensure(ctx, network)
// 	if err != nil {
// 		return network, errors.Wrapf(err, "fail to ensure network %s", network)
// 	}

// 	return network, nil
// }

// func (c *repository) Exists(ctx context.Context, id string) (types.Network, bool, error) {
// 	network := types.Network{
// 		ID: id,
// 	}
// 	if id == "" {
// 		return network, false, nil
// 	}

// 	err := c.store.Get(ctx, network.StorageKey(), false, &network)
// 	if err != nil && err != store.ErrNotFound {
// 		return network, false, errors.Wrapf(err, "fail to get network %s from store", network.Name)
// 	}
// 	if err == store.ErrNotFound {
// 		return network, false, nil
// 	}
// 	return network, true, err
// }

// func (c *repository) new(ctx context.Context, params params.CreateNetworkParams) (types.Network, error) {
// 	log := logger.Get(ctx)
// 	uuid := uuid.NewRandom().String()

// 	iprange := DefaultIPRange
// 	if params.IPRange != "" {
// 		_, _, err := net.ParseCIDR(params.IPRange)
// 		if err != nil {
// 			return types.Network{}, errors.Wrapf(err, "invalid IP CIDR")
// 		}
// 		iprange = params.IPRange
// 	}

// 	allocator := ipallocator.New(c.config, c.store, uuid, ipallocator.WithIPRange(iprange))
// 	ip, mask, err := allocator.AllocateIP(ctx)
// 	if err != nil {
// 		return types.Network{}, errors.Wrapf(err, "fail to allocate gateway IP")
// 	}

// 	log.Infof("gateway IP allocated: %s/%d", ip, mask)

// 	network := types.Network{
// 		ID:        uuid,
// 		IPRange:   iprange,
// 		Gateway:   fmt.Sprintf("%s/%d", ip.String(), mask),
// 		CreatedAt: time.Now(),
// 		Name:      params.Name,
// 		Type:      params.Type,
// 		NSHandlePath: filepath.Join(
// 			c.config.NetnsPath, fmt.Sprintf("%s%s", c.config.NetnsPrefix, uuid),
// 		),
// 	}

// 	vniGen := overlay.NewVNIGenerator(ctx, c.config, c.store)

// 	switch network.Type {
// 	case types.OverlayNetworkType:
// 		err := vniGen.Lock(ctx)
// 		if err != nil {
// 			return network, errors.Wrapf(err, "fail to lock VNI generator for %s", network)
// 		}
// 		vni, err := vniGen.Generate(ctx)
// 		if err != nil {
// 			return network, errors.Wrapf(err, "fail to generate VNI for %s", network)
// 		}

// 		log.Debugf("vni is %v", vni)
// 		network.VxLANVNI = vni
// 	default:
// 		return network, errors.New("invalid network type for init")
// 	}

// 	err = c.store.Set(ctx, network.StorageKey(), &network)
// 	if err != nil {
// 		return network, errors.Wrapf(err, "fail to get network %s from store", network)
// 	}

// 	if vniGen != nil {
// 		err := vniGen.Unlock(ctx)
// 		if err != nil {
// 			log.WithError(err).Errorf("fail to unlock VNI generator for %s", network)
// 		}
// 	}
// 	return network, nil
// }
