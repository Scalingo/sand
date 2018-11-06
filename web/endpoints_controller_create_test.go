package web

import (
	"errors"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/Scalingo/sand/api/params"
	"github.com/Scalingo/sand/api/types"
	"github.com/Scalingo/sand/ipallocator"
	"github.com/Scalingo/sand/test/mocks/endpointmock"
	"github.com/Scalingo/sand/test/mocks/ipallocatormock"
	"github.com/Scalingo/sand/test/mocks/networkmock"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestEndpointsController_Create(t *testing.T) {
	cases := []struct {
		Name                     string
		Path                     string
		Method                   string
		Body                     string
		Status                   int
		Error                    string
		ExpectNetworkRepository  func(*networkmock.MockRepository)
		ExpectEndpointRepository func(*endpointmock.MockRepository)
		ExpectIPAllocator        func(*ipallocatormock.MockIPAllocator)
	}{
		{
			Name:   "invalid JSON should return 400",
			Path:   "/endpoints",
			Method: "POST",
			Body:   `{`,
			Status: 400,
			Error:  "invalid JSON",
		}, {
			Name:   "unexisting network id should return 404",
			Path:   "/endpoints",
			Method: "POST",
			Body:   `{"network_id": "1"}`,
			Status: 404,
			Error:  "not found",
			ExpectNetworkRepository: func(r *networkmock.MockRepository) {
				r.EXPECT().Exists(gomock.Any(), "1").Return(types.Network{}, false, nil)
			},
		}, {
			Name:   "error finding network should return error",
			Path:   "/endpoints",
			Method: "POST",
			Body:   `{"network_id": "1"}`,
			Error:  "network repo error",
			ExpectNetworkRepository: func(r *networkmock.MockRepository) {
				r.EXPECT().Exists(gomock.Any(), "1").Return(types.Network{}, false, errors.New("network repo error"))
			},
		}, {
			Name:   "error if network ensure fails",
			Path:   "/endpoints",
			Method: "POST",
			Body:   `{"network_id": "1"}`,
			Error:  "fail to ensure network",
			ExpectIPAllocator: func(m *ipallocatormock.MockIPAllocator) {
				m.EXPECT().AllocateIP(gomock.Any(), "1", ipallocator.AllocateIPOpts{
					Address: "",
				})
			},
			ExpectNetworkRepository: func(r *networkmock.MockRepository) {
				network := types.Network{ID: "1"}
				r.EXPECT().Exists(gomock.Any(), "1").Return(network, true, nil)
				r.EXPECT().Ensure(gomock.Any(), network).Return(errors.New("fail to ensure network"))
			},
		}, {
			Name:   "error if endpoint creation fails",
			Path:   "/endpoints",
			Method: "POST",
			Body:   `{"network_id": "1", "activate": true, "activate_params": { "ns_handle_path": "/proc/self/ns/net"}}`,
			Error:  "fail to create endpoint",
			ExpectIPAllocator: func(m *ipallocatormock.MockIPAllocator) {
				m.EXPECT().AllocateIP(gomock.Any(), "1", ipallocator.AllocateIPOpts{
					Address: "",
				})
			},
			ExpectNetworkRepository: func(r *networkmock.MockRepository) {
				network := types.Network{ID: "1"}
				r.EXPECT().Exists(gomock.Any(), "1").Return(network, true, nil)
				r.EXPECT().Ensure(gomock.Any(), network).Return(nil)
			},
			ExpectEndpointRepository: func(r *endpointmock.MockRepository) {
				network := types.Network{ID: "1"}
				params := params.EndpointCreate{
					NetworkID: network.ID,
					Activate:  true,
					ActivateParams: params.EndpointActivate{
						NSHandlePath: "/proc/self/ns/net",
						MoveVeth:     true,
						SetAddr:      true,
					},
				}
				r.EXPECT().Create(gomock.Any(), network, params).Return(types.Endpoint{}, errors.New("fail to create endpoint"))
			},
		},
	}

	for _, c := range cases {
		t.Run(c.Name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			networkRepo := networkmock.NewMockRepository(ctrl)
			endpointRepo := endpointmock.NewMockRepository(ctrl)
			ipallocator := ipallocatormock.NewMockIPAllocator(ctrl)

			controller := EndpointsController{
				EndpointRepository: endpointRepo,
				NetworkRepository:  networkRepo,
				IPAllocator:        ipallocator,
			}

			if c.ExpectNetworkRepository != nil {
				c.ExpectNetworkRepository(networkRepo)
			}

			if c.ExpectEndpointRepository != nil {
				c.ExpectEndpointRepository(endpointRepo)
			}

			if c.ExpectIPAllocator != nil {
				c.ExpectIPAllocator(ipallocator)
			}

			body := strings.NewReader(c.Body)
			r := httptest.NewRequest(c.Method, c.Path, body)
			w := httptest.NewRecorder()

			err := controller.Create(w, r, map[string]string{})
			if c.Status != 0 {
				assert.Equal(t, w.Code, c.Status)
			}
			if c.Error != "" {
				require.Error(t, err)
				require.Contains(t, err.Error(), c.Error)
				return
			}
			require.NoError(t, err)
		})
	}
}
