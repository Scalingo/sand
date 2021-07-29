package store

import (
	"context"
	"testing"
	"time"

	"github.com/Scalingo/sand/config"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.etcd.io/etcd/api/v3/mvccpb"
	clientv3 "go.etcd.io/etcd/client/v3"
)

func TestWatcher_Register(t *testing.T) {
	cases := map[string]struct {
		registrationKey string
		expect          func(t *testing.T, w Watcher, in chan clientv3.WatchResponse, r Registration)
	}{
		"it should get an event impacting a subkey": {
			registrationKey: "/prefix/subkey1",
			expect: func(t *testing.T, w Watcher, in chan clientv3.WatchResponse, r Registration) {
				in <- clientv3.WatchResponse{
					Events: []*clientv3.Event{{
						Kv:   &mvccpb.KeyValue{Key: []byte("/sc-net/prefix/subkey1/key1")},
						Type: mvccpb.PUT,
					}},
				}
				event := <-r.EventChan()
				assert.Equal(t, string(event.Kv.Key), "/sc-net/prefix/subkey1/key1")
				assert.Equal(t, event.Type, mvccpb.PUT)
			},
		},
		"it should not get an event after unregistration": {
			registrationKey: "/prefix/subkey2",
			expect: func(t *testing.T, w Watcher, in chan clientv3.WatchResponse, r Registration) {
				r.Unregister()
				// buffered chan, non blocking operation
				in <- clientv3.WatchResponse{
					Events: []*clientv3.Event{{Kv: &mvccpb.KeyValue{}}},
				}
				select {
				case _, ok := <-r.EventChan():
					if ok {
						require.Fail(t, "should not get an event")
					}
				}
			},
		},
		"it should log an error and keep on listening until the chan is closed": {
			registrationKey: "/prefix/subkey3",
			expect: func(t *testing.T, w Watcher, in chan clientv3.WatchResponse, r Registration) {
				in <- clientv3.WatchResponse{
					Canceled: true,
				}
				select {
				case <-r.EventChan():
					t.Error("nothing should get out of this")
				default:
				}
			},
		},
	}

	for title, c := range cases {
		t.Run(title, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			etcdWatcher := NewMockEtcdWatcher(ctrl)
			incomingEvents := make(chan clientv3.WatchResponse, 1)

			etcdWatcher.EXPECT().WatchChan().Return(clientv3.WatchChan(incomingEvents))
			etcdWatcher.EXPECT().Close().Return(nil)

			config, err := config.Build()
			require.NoError(t, err)

			Watcher, err := NewWatcher(
				context.Background(), config,
				WithEtcdWatcher(etcdWatcher),
				WithPrefix("/prefix"),
			)
			require.NoError(t, err)
			defer Watcher.Close()

			r, err := Watcher.Register(c.registrationKey)
			require.NoError(t, err)

			// Ensure the watcher has started watching etcd events, sleep to schedule the goroutine
			time.Sleep(10 * time.Millisecond)

			c.expect(t, Watcher, incomingEvents, r)
		})
	}
}
