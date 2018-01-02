package overlay

import (
	"context"
	"testing"

	"github.com/Scalingo/sand/api/types"
	"github.com/Scalingo/sand/config"
	"github.com/Scalingo/sand/test/mocks/network/netmanagermock"
	"github.com/Scalingo/sand/test/mocks/storemock"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/require"
)

func TestListener_Add(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	nm := netmanagermock.NewMockNetManager(ctrl)
	store := storemock.NewMockStore(ctrl)
	config, err := config.Build()
	require.NoError(t, err)
	listener := NewNetworkEndpointListener(config, store)

	network := types.Network{ID: "1"}

	w := storemock.NewMockWatcher(ctrl)

	store.EXPECT().Watch(gomock.Any(), network.EndpointsStorageKey("")).Return(w, nil)

	err = listener.Add(context.Background(), nm, network)
	require.NoError(t, err)
}

func TestListener_Remove(t *testing.T) {

}
