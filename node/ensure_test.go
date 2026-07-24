package node

import (
	"os"
	"testing"

	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	"github.com/Scalingo/sand/api/params"
	"github.com/Scalingo/sand/api/types"
	"github.com/Scalingo/sand/config"
	"github.com/Scalingo/sand/test/mocks/endpointmock"
	"github.com/Scalingo/sand/test/mocks/networkmock"
)

func TestEnsureNetworkEndpoints(t *testing.T) {
	cases := []struct {
		Name                     string
		ExpectNetworkRepository  func(*networkmock.MockRepository)
		ExpectEndpointRepository func(*endpointmock.MockRepository)
	}{
		{
			Name: "activate endpoints on current node",
			ExpectEndpointRepository: func(r *endpointmock.MockRepository) {
				endpoint := types.Endpoint{
					ID:              "endpoint-1",
					NetworkID:       "network-1",
					Active:          true,
					TargetNetnsPath: "/proc/1/ns/net",
				}
				network := types.Network{ID: "network-1"}

				r.EXPECT().List(gomock.Any(), map[string]string{"hostname": "node-1"}).Return([]types.Endpoint{endpoint}, nil)
				r.EXPECT().Activate(gomock.Any(), network, endpoint, params.EndpointActivate{
					NSHandlePath: "/proc/1/ns/net",
					SetAddr:      true,
					MoveVeth:     true,
				}).Return(endpoint, nil)
			},
			ExpectNetworkRepository: func(r *networkmock.MockRepository) {
				network := types.Network{ID: "network-1"}

				r.EXPECT().Exists(gomock.Any(), "network-1").Return(network, true, nil)
				r.EXPECT().Ensure(gomock.Any(), network).Return(nil)
			},
		}, {
			Name: "skip inactive endpoints",
			ExpectEndpointRepository: func(r *endpointmock.MockRepository) {
				r.EXPECT().List(gomock.Any(), map[string]string{"hostname": "node-1"}).Return([]types.Endpoint{{
					ID:        "endpoint-1",
					NetworkID: "network-1",
					Active:    false,
				}}, nil)
			},
		}, {
			Name: "deactivate endpoint when netns no longer exists",
			ExpectEndpointRepository: func(r *endpointmock.MockRepository) {
				endpoint := types.Endpoint{
					ID:              "endpoint-1",
					NetworkID:       "network-1",
					Active:          true,
					TargetNetnsPath: "/proc/1/ns/net",
				}
				network := types.Network{ID: "network-1"}

				r.EXPECT().List(gomock.Any(), map[string]string{"hostname": "node-1"}).Return([]types.Endpoint{endpoint}, nil)
				r.EXPECT().Activate(gomock.Any(), network, endpoint, params.EndpointActivate{
					NSHandlePath: "/proc/1/ns/net",
					SetAddr:      true,
					MoveVeth:     true,
				}).Return(types.Endpoint{}, os.ErrNotExist)
				r.EXPECT().Deactivate(gomock.Any(), network, endpoint).Return(endpoint, nil)
			},
			ExpectNetworkRepository: func(r *networkmock.MockRepository) {
				network := types.Network{ID: "network-1"}

				r.EXPECT().Exists(gomock.Any(), "network-1").Return(network, true, nil)
				r.EXPECT().Ensure(gomock.Any(), network).Return(nil)
			},
		},
	}

	for _, c := range cases {
		t.Run(c.Name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			networkRepo := networkmock.NewMockRepository(ctrl)
			endpointRepo := endpointmock.NewMockRepository(ctrl)

			if c.ExpectNetworkRepository != nil {
				c.ExpectNetworkRepository(networkRepo)
			}

			if c.ExpectEndpointRepository != nil {
				c.ExpectEndpointRepository(endpointRepo)
			}

			err := EnsureNetworkEndpoints(t.Context(), &config.Config{PeerHostname: "node-1"}, networkRepo, endpointRepo)
			require.NoError(t, err)
		})
	}
}
