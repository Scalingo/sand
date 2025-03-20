package network

import (
	"context"
	"errors"
	"reflect"
	"testing"
	"time"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/Scalingo/sand/api/types"
	"github.com/Scalingo/sand/config"
	"github.com/Scalingo/sand/network/netmanager"
	"github.com/Scalingo/sand/store/storemock"
	"github.com/Scalingo/sand/test/mocks/network/netmanagermock"
)

func TestRepository_Ensure(t *testing.T) {
	expectNetManager := func(err error) func(t *testing.T, m *netmanagermock.MockNetManager, n types.Network) {
		return func(t *testing.T, m *netmanagermock.MockNetManager, n types.Network) {
			m.EXPECT().Ensure(gomock.Any(), n).Return(nil)
			m.EXPECT().EnsureEndpointsNeigh(gomock.Any(), n, gomock.Any()).Do(func(ctx context.Context, n types.Network, eps []types.Endpoint) {
				require.Len(t, eps, 1)
				require.Equal(t, eps[0].ID, "ep-1")
			}).Return(err)
			if err == nil {
				m.EXPECT().ListenNetworkChange(gomock.Any(), n).Return(nil)
			}
		}
	}

	cases := []struct {
		Name             string
		Network          func() types.Network
		Error            string
		ExpectNetManager func(*testing.T, *netmanagermock.MockNetManager, types.Network)
		ExpectStore      func(*testing.T, *storemock.MockStore, types.Network)
	}{
		{
			Name: "network with unknown type should return an error",
			Network: func() types.Network {
				return types.Network{Type: types.NetworkType("unknown")}
			},
			Error: "invalid",
		}, {
			Name: "overlay: should return error if manager Ensure fails",
			ExpectNetManager: func(t *testing.T, m *netmanagermock.MockNetManager, n types.Network) {
				m.EXPECT().Ensure(gomock.Any(), n).Return(errors.New("fail to ensure network"))
			},
			Error: "fail to ensure",
		}, {
			Name: "overlay: should return error if storage fails to retrieve list of endpoints",
			ExpectNetManager: func(t *testing.T, m *netmanagermock.MockNetManager, n types.Network) {
				m.EXPECT().Ensure(gomock.Any(), n).Return(nil)
			},
			ExpectStore: func(t *testing.T, m *storemock.MockStore, n types.Network) {
				m.EXPECT().Get(
					gomock.Any(), n.EndpointsStorageKey(""), true, gomock.Any(),
				).Return(errors.New("fail to get endpoints"))
			},
			Error: "fail to get endpoints",
		}, {
			Name:             "overlay: if there are more than 1 endpoint, add neighbors",
			ExpectNetManager: expectNetManager(errors.New("fail to add neighbors")),
			ExpectStore: func(t *testing.T, m *storemock.MockStore, n types.Network) {
				m.EXPECT().Get(
					gomock.Any(), n.EndpointsStorageKey(""), true, gomock.Any(),
				).Do(
					func(ctx context.Context, key string, recursive bool, data interface{}) {
						reflect.ValueOf(data).Elem().Set(reflect.ValueOf([]types.Endpoint{{ID: "ep-1"}}))
					},
				).Return(nil)
			},
			Error: "fail to add neighbors",
		}, {
			Name:             "overlay: it should add entries in the store",
			ExpectNetManager: expectNetManager(nil),
			ExpectStore: func(t *testing.T, m *storemock.MockStore, n types.Network) {
				m.EXPECT().Get(
					gomock.Any(), n.EndpointsStorageKey(""), true, gomock.Any(),
				).Do(
					func(ctx context.Context, key string, recursive bool, data interface{}) {
						reflect.ValueOf(data).Elem().Set(reflect.ValueOf([]types.Endpoint{{ID: "ep-1"}}))
					},
				).Return(nil)
				for _, key := range []string{"/nodes/test-hostname/networks/1", "/nodes-networks/1/test-hostname"} {
					m.EXPECT().Set(
						gomock.Any(), key, gomock.Any(),
					).Do(func(ctx context.Context, key string, data interface{}) {
						m, ok := data.(map[string]interface{})
						assert.True(t, ok)
						assert.Equal(t, "1", m["id"].(string))
						assert.IsType(t, time.Now(), m["created_at"])
					}).Return(nil)
				}
			},
		},
	}

	for _, c := range cases {
		t.Run(c.Name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			config, err := config.Build()
			config.PeerHostname = "test-hostname"

			require.NoError(t, err)

			omanager := netmanagermock.NewMockNetManager(ctrl)
			managers := netmanager.NewManagerMap()
			managers.Set(types.OverlayNetworkType, omanager)
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
				c.ExpectNetManager(t, omanager, network)
			}

			if c.ExpectStore != nil {
				c.ExpectStore(t, store, network)
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
