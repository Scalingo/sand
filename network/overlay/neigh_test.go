package overlay

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/Scalingo/sand/api/types"
	"github.com/Scalingo/sand/config"
)

func TestManager_EnsureEndpointsNeigh_IgnoresInactiveEndpoints(t *testing.T) {
	m := manager{
		config: &config.Config{PeerIP: "192.0.2.10"},
	}

	err := m.EnsureEndpointsNeigh(t.Context(), types.Network{NSHandlePath: "/does/not/exist"}, []types.Endpoint{
		{
			ID:           "active-endpoint",
			Active:       true,
			HostIP:       "192.0.2.10",
			TargetVethIP: "10.0.0.5/24",
		},
		{
			ID:           "inactive-endpoint",
			Active:       false,
			HostIP:       "192.0.2.20",
			TargetVethIP: "10.0.0.5/24",
		},
	})

	require.NoError(t, err)
}
