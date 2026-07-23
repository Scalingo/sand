package overlay

import (
	"net"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/vishvananda/netlink"
	"golang.org/x/sys/unix"

	"github.com/Scalingo/sand/api/types"
	"github.com/Scalingo/sand/config"
)

func TestManager_AddEndpointNeigh_IgnoresInactiveEndpoints(t *testing.T) {
	m := manager{
		config: &config.Config{PeerIP: "192.0.2.10"},
	}

	results, err := m.AddEndpointNeigh(t.Context(), types.Network{NSHandlePath: "/does/not/exist"}, types.Endpoint{
		ID:           "inactive-endpoint",
		Active:       false,
		HostIP:       "192.0.2.20",
		TargetVethIP: "10.0.0.5/24",
	})

	require.NoError(t, err)
	require.False(t, results.AddedARPEntry)
	require.False(t, results.AddedFDBEntry)
}

func TestManager_AddEndpointNeigh_IgnoresLocalEndpoints(t *testing.T) {
	m := manager{
		config: &config.Config{PeerIP: "192.0.2.10"},
	}

	results, err := m.AddEndpointNeigh(t.Context(), types.Network{NSHandlePath: "/does/not/exist"}, types.Endpoint{
		ID:           "local-endpoint",
		Active:       true,
		HostIP:       "192.0.2.10",
		TargetVethIP: "10.0.0.5/24",
	})

	require.NoError(t, err)
	require.False(t, results.AddedARPEntry)
	require.False(t, results.AddedFDBEntry)
}

func TestNeighExists(t *testing.T) {
	mac, err := net.ParseMAC("02:42:ac:11:00:02")
	require.NoError(t, err)

	expected := netlink.Neigh{
		LinkIndex:    12,
		Family:       unix.AF_BRIDGE,
		State:        netlink.NUD_PERMANENT,
		Flags:        netlink.NTF_SELF,
		IP:           net.ParseIP("192.0.2.10"),
		HardwareAddr: mac,
	}

	require.True(t, neighExists([]netlink.Neigh{expected}, expected))

	staleMAC, err := net.ParseMAC("02:42:ac:11:00:03")
	require.NoError(t, err)
	stale := expected
	stale.HardwareAddr = staleMAC

	require.False(t, neighExists([]netlink.Neigh{stale}, expected))
}
