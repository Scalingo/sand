package web

import (
	"errors"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/Scalingo/sand/api/params"
	"github.com/Scalingo/sand/api/types"
	"github.com/Scalingo/sand/test/mocks/endpointmock"
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
	}{
		{
			Name:   "invalid JSON should return 400",
			Path:   "/endpoints",
			Method: "POST",
			Body:   `{`,
			Status: 400,
			Error:  "invalid JSON",
		}, {
			Name:   "missing ns_handle_path should return 400",
			Path:   "/endpoints",
			Method: "POST",
			Body:   `{"network_id": "1"}`,
			Status: 400,
			Error:  "missing ns_handle_path",
		}, {
			Name:   "un-stat-able ns_handle_path should return 400",
			Path:   "/endpoints",
			Method: "POST",
			Body:   `{"network_id": "1", "ns_handle_path": "/tmp/random-1234567890"}`,
			Status: 400,
			Error:  "no such file or directory",
		}, {
			Name:   "unexisting network id should return 404",
			Path:   "/endpoints",
			Method: "POST",
			Body:   `{"network_id": "1", "ns_handle_path": "/proc/self/ns/net"}`,
			Status: 404,
			Error:  "not found",
			ExpectNetworkRepository: func(r *networkmock.MockRepository) {
				r.EXPECT().Exists(gomock.Any(), "1").Return(types.Network{}, false, nil)
			},
		}, {
			Name:   "error finding network should return error",
			Path:   "/endpoints",
			Method: "POST",
			Body:   `{"network_id": "1", "ns_handle_path": "/proc/self/ns/net"}`,
			Error:  "network repo error",
			ExpectNetworkRepository: func(r *networkmock.MockRepository) {
				r.EXPECT().Exists(gomock.Any(), "1").Return(types.Network{}, false, errors.New("network repo error"))
			},
		}, {
			Name:   "error if network ensure fails",
			Path:   "/endpoints",
			Method: "POST",
			Body:   `{"network_id": "1", "ns_handle_path": "/proc/self/ns/net"}`,
			Error:  "fail to ensure network",
			ExpectNetworkRepository: func(r *networkmock.MockRepository) {
				network := types.Network{ID: "1"}
				r.EXPECT().Exists(gomock.Any(), "1").Return(network, true, nil)
				r.EXPECT().Ensure(gomock.Any(), network).Return(errors.New("fail to ensure network"))
			},
		}, {
			Name:   "error if endpoint creation fails",
			Path:   "/endpoints",
			Method: "POST",
			Body:   `{"network_id": "1", "ns_handle_path": "/proc/self/ns/net"}`,
			Error:  "fail to create endpoint",
			ExpectNetworkRepository: func(r *networkmock.MockRepository) {
				network := types.Network{ID: "1"}
				r.EXPECT().Exists(gomock.Any(), "1").Return(network, true, nil)
				r.EXPECT().Ensure(gomock.Any(), network).Return(nil)
			},
			ExpectEndpointRepository: func(r *endpointmock.MockRepository) {
				network := types.Network{ID: "1"}
				params := params.EndpointCreate{NetworkID: network.ID, NSHandlePath: "/proc/self/ns/net"}
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

			controller := EndpointsController{
				EndpointRepository: endpointRepo, NetworkRepository: networkRepo,
			}

			if c.ExpectNetworkRepository != nil {
				c.ExpectNetworkRepository(networkRepo)
			}

			if c.ExpectEndpointRepository != nil {
				c.ExpectEndpointRepository(endpointRepo)
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
