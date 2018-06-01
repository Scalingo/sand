package overlay

import (
	"context"
	"testing"
	"time"

	"github.com/Scalingo/sand/api/types"
	"github.com/Scalingo/sand/config"
	"github.com/Scalingo/sand/store"
	"github.com/Scalingo/sand/test/mocks/network/netmanagermock"
	"github.com/Scalingo/sand/test/mocks/storemock"
	"github.com/coreos/etcd/clientv3"
	"github.com/coreos/etcd/mvcc/mvccpb"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestListener_Add(t *testing.T) {
	cases := []struct {
		Name             string
		Error            string
		ExpectStore      func(m *storemock.MockStore, network types.Network, watcher store.Watcher)
		ExpectWatcher    func(m *storemock.MockWatcher)
		ExpectNetManager func(m *netmanagermock.MockNetManager, n types.Network)
		ExpectDone       bool
	}{
		{
			Name: "it should start listening to message and stop when there is no more message",
			ExpectStore: func(m *storemock.MockStore, network types.Network, watcher store.Watcher) {
				m.EXPECT().Watch(gomock.Any(), network.EndpointsStorageKey("")).Return(watcher, nil)
			},
			ExpectWatcher: func(m *storemock.MockWatcher) {
				m.EXPECT().NextResponse().Return(clientv3.WatchResponse{}, false)
			},
			ExpectDone: true,
		}, {
			Name:       "it should handle PUT message with an Endpoint and add it as neighbor",
			ExpectDone: true,
			ExpectStore: func(m *storemock.MockStore, network types.Network, watcher store.Watcher) {
				m.EXPECT().Watch(gomock.Any(), network.EndpointsStorageKey("")).Return(watcher, nil)
			},
			ExpectWatcher: func(m *storemock.MockWatcher) {
				m.EXPECT().NextResponse().Return(clientv3.WatchResponse{
					Events: []*clientv3.Event{{
						Type: mvccpb.PUT,
						Kv: &mvccpb.KeyValue{
							Value: []byte(`{
								"id": "1",
								"target_veth_ip": "10.0.0.1",
								"hostname": "src-node.example.com"
							}`),
						},
					}},
				}, true)
				m.EXPECT().NextResponse().Return(clientv3.WatchResponse{}, false)
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
			watcher := storemock.NewMockWatcher(ctrl)

			config, err := config.Build()
			require.NoError(t, err)

			listener := NewNetworkEndpointListener(context.Background(), config, store)
			network := types.Network{ID: "1"}

			if c.ExpectStore != nil {
				c.ExpectStore(store, network, watcher)
			}

			if c.ExpectWatcher != nil {
				c.ExpectWatcher(watcher)
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
