package overlay

import (
	"context"
	"testing"
	"time"

	"go.etcd.io/etcd/api/v3/mvccpb"
	clientv3 "go.etcd.io/etcd/client/v3"

	"github.com/Scalingo/sand/api/types"
	"github.com/Scalingo/sand/config"
	"github.com/Scalingo/sand/network/overlay/overlaymock"
	"github.com/Scalingo/sand/store/storemock"
	"github.com/Scalingo/sand/test/mocks/network/netmanagermock"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestListener_Add(t *testing.T) {
	cases := []struct {
		Name               string
		Error              string
		ExpectStore        func(m *storemock.MockStore, network types.Network, registrar Registrar)
		ExpectRegistration func(r *storemock.MockRegistration)
		ExpectNetManager   func(m *netmanagermock.MockNetManager, n types.Network)
		ExpectDone         bool
	}{
		{
			Name:       "it should start listening to message and stop when there is no more message",
			ExpectDone: true,
			ExpectRegistration: func(r *storemock.MockRegistration) {
				c := make(chan *clientv3.Event)
				close(c)
				r.EXPECT().EventChan().Return(c)
			},
		}, {
			Name:       "it should handle PUT message with an Endpoint and add it as neighbor",
			ExpectDone: true,
			ExpectRegistration: func(r *storemock.MockRegistration) {
				c := make(chan *clientv3.Event, 1)
				c <- &clientv3.Event{
					Type: mvccpb.PUT,
					Kv: &mvccpb.KeyValue{
						Value: []byte(`{
							"id": "1",
							"target_veth_ip": "10.0.0.1",
							"hostname": "src-node.example.com"
						}`),
					},
				}
				close(c)
				r.EXPECT().EventChan().Return(c)
			},
			ExpectNetManager: func(m *netmanagermock.MockNetManager, n types.Network) {
				m.EXPECT().AddEndpointNeigh(gomock.Any(), n, types.Endpoint{
					ID: "1", TargetVethIP: "10.0.0.1", Hostname: "src-node.example.com",
				}).Return(nil)
			},
		},
	}
	for _, c := range cases {
		t.Run(c.Name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()
			nm := netmanagermock.NewMockNetManager(ctrl)
			store := storemock.NewMockStore(ctrl)
			registrar := overlaymock.NewMockRegistrar(ctrl)
			registration := storemock.NewMockRegistration(ctrl)

			config, err := config.Build()
			require.NoError(t, err)

			network := types.Network{ID: "1"}
			registrar.EXPECT().Register("/network-endpoints/1").Return(registration, nil)

			listener := NewNetworkEndpointListener(context.Background(), config, registrar, store)

			if c.ExpectStore != nil {
				c.ExpectStore(store, network, registrar)
			}

			if c.ExpectRegistration != nil {
				c.ExpectRegistration(registration)
			}

			if c.ExpectNetManager != nil {
				c.ExpectNetManager(nm, network)
			}

			done, err := listener.Add(context.Background(), nm, network)
			if c.Error != "" {
				require.Error(t, err)
				assert.Contains(t, err.Error(), c.Error)
				return
			}

			require.NoError(t, err)
			if c.ExpectDone {
				select {
				case <-time.NewTimer(time.Second).C:
					require.Fail(t, "should not timeout")
				case <-done:
				}
			}
		})
	}
}

func TestListener_Remove(t *testing.T) {

}
