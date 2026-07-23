package overlay

import (
	"context"
	"fmt"
	"net"

	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"github.com/vishvananda/netlink"
	"github.com/vishvananda/netns"
	"golang.org/x/sys/unix"

	"github.com/Scalingo/go-utils/logger"
	"github.com/Scalingo/sand/api/types"
	"github.com/Scalingo/sand/network/netmanager"
)

type neighAction string

const (
	neighActionSet neighAction = "set"
	neighActionDel neighAction = "delete"
)

func (m manager) EnsureEndpointsNeigh(ctx context.Context, network types.Network, endpoints []types.Endpoint) (netmanager.EnsureEndpointsNeighResults, error) {
	results := netmanager.EnsureEndpointsNeighResults{}

	nsfd, err := netns.GetFromPath(network.NSHandlePath)
	if err != nil {
		return results, errors.Wrapf(err, "fail to get namespace handler")
	}
	defer nsfd.Close()

	nlh, err := netlink.NewHandleAt(nsfd, unix.NETLINK_ROUTE)
	if err != nil {
		return results, errors.Wrapf(err, "fail to get netlink handler of netns")
	}
	defer nlh.Delete()

	link, err := nlh.LinkByName(VxLANInNSName)
	if err != nil {
		return results, errors.Wrapf(err, "fail to get vxlan interface")
	}

	expectedARPEntries, expectedFDBEntries, err := m.expectedNeighs(endpoints, link.Attrs().Index)
	if err != nil {
		return results, err
	}

	arpEntries, err := nlh.NeighList(link.Attrs().Index, netlink.FAMILY_V4)
	if err != nil {
		return results, errors.Wrapf(err, "list ARP entries")
	}
	fdbEntries, err := nlh.NeighList(link.Attrs().Index, unix.AF_BRIDGE)
	if err != nil {
		return results, errors.Wrapf(err, "list FDB entries")
	}

	results.AddedARPEntries, results.RemovedARPEntries, err = reconcileNeighs(nlh, arpEntries, expectedARPEntries, isManagedARPEntry)
	if err != nil {
		return results, errors.Wrapf(err, "reconcile ARP entries")
	}
	results.AddedFDBEntries, results.RemovedFDBEntries, err = reconcileNeighs(nlh, fdbEntries, expectedFDBEntries, isManagedFDBEntry)
	if err != nil {
		return results, errors.Wrapf(err, "reconcile FDB entries")
	}

	return results, nil
}

func (m manager) expectedNeighs(endpoints []types.Endpoint, linkIndex int) ([]netlink.Neigh, []netlink.Neigh, error) {
	arpEntries := make([]netlink.Neigh, 0, len(endpoints))
	fdbEntries := make([]netlink.Neigh, 0, len(endpoints))

	for _, endpoint := range endpoints {
		if !endpoint.Active || endpoint.HostIP == m.config.GetPeerIP() {
			continue
		}

		ip, _, err := net.ParseCIDR(endpoint.TargetVethIP)
		if err != nil {
			return nil, nil, errors.Wrapf(err, "fail to parse IP of %v '%s'", endpoint.TargetVethName, endpoint.TargetVethIP)
		}
		mac, err := net.ParseMAC(endpoint.TargetVethMAC)
		if err != nil {
			return nil, nil, errors.Wrapf(err, "fail to parse MAC of %v '%s'", endpoint.TargetVethName, endpoint.TargetVethMAC)
		}
		vtepIP := net.ParseIP(endpoint.HostIP)
		if vtepIP == nil {
			return nil, nil, errors.Errorf("fail to parse endpoint host IP (VTEP IP) '%s'", endpoint.HostIP)
		}

		arpEntries = append(arpEntries, netlink.Neigh{
			IP:           ip,
			HardwareAddr: mac,
			State:        netlink.NUD_PERMANENT,
			LinkIndex:    linkIndex,
			Family:       netlink.FAMILY_V4,
		})
		fdbEntries = append(fdbEntries, netlink.Neigh{
			IP:           vtepIP,
			HardwareAddr: mac,
			State:        netlink.NUD_PERMANENT,
			LinkIndex:    linkIndex,
			Family:       unix.AF_BRIDGE,
			Flags:        netlink.NTF_SELF,
		})
	}

	return arpEntries, fdbEntries, nil
}

func (m manager) AddEndpointNeigh(ctx context.Context, network types.Network, endpoint types.Endpoint) (netmanager.EndpointNeighResults, error) {
	ctx, log := logger.WithFieldsToCtx(ctx, logrus.Fields{
		"neighbor_action":          "add",
		"endpoint_id":              endpoint.ID,
		"endpoint_target_ip":       endpoint.TargetVethIP,
		"endpoint_target_hostname": endpoint.Hostname,
	})
	if !endpoint.Active {
		log.Info("Skip inactive endpoint ARP/FDB replay")
		return netmanager.EndpointNeighResults{}, nil
	}

	return m.endpointNeighAction(ctx, network, endpoint, neighActionSet)
}

func (m manager) RemoveEndpointNeigh(ctx context.Context, network types.Network, endpoint types.Endpoint) (netmanager.EndpointNeighResults, error) {
	ctx = logger.ToCtx(ctx, logger.Get(ctx).WithField("neighbor_action", "delete"))
	return m.endpointNeighAction(ctx, network, endpoint, neighActionDel)
}

func (m manager) endpointNeighAction(ctx context.Context, network types.Network, endpoint types.Endpoint, action neighAction) (netmanager.EndpointNeighResults, error) {
	log := logger.Get(ctx)

	// No rule to add for endpoint located on the current server
	if endpoint.HostIP == m.config.GetPeerIP() {
		return netmanager.EndpointNeighResults{}, nil
	}

	log.Info("change endpoint ARP/FDB rules")

	nsfd, err := netns.GetFromPath(network.NSHandlePath)
	if err != nil {
		return netmanager.EndpointNeighResults{}, errors.Wrapf(err, "fail to get namespace handler")
	}
	defer nsfd.Close()

	nlh, err := netlink.NewHandleAt(nsfd, unix.NETLINK_ROUTE)
	if err != nil {
		return netmanager.EndpointNeighResults{}, errors.Wrapf(err, "fail to get netlink handler of netns")
	}
	defer nlh.Delete()

	link, err := nlh.LinkByName(VxLANInNSName)
	if err != nil {
		return netmanager.EndpointNeighResults{}, errors.Wrapf(err, "fail to get vxlan interface")
	}

	ip, _, err := net.ParseCIDR(endpoint.TargetVethIP)
	if err != nil {
		return netmanager.EndpointNeighResults{}, errors.Wrapf(err, "fail to parse IP of %v '%s'", endpoint.TargetVethName, endpoint.TargetVethIP)
	}
	mac, err := net.ParseMAC(endpoint.TargetVethMAC)
	if err != nil {
		return netmanager.EndpointNeighResults{}, errors.Wrapf(err, "fail to parse MAC of %v '%s'", endpoint.TargetVethName, endpoint.TargetVethMAC)
	}
	vtepIP := net.ParseIP(endpoint.HostIP)
	if vtepIP == nil {
		return netmanager.EndpointNeighResults{}, errors.Errorf("fail to parse endpoint host IP (VTEP IP) '%s'", endpoint.HostIP)
	}

	nlnh := &netlink.Neigh{
		IP:           ip,
		HardwareAddr: mac,
		State:        netlink.NUD_PERMANENT,
		LinkIndex:    link.Attrs().Index,
		Family:       netlink.FAMILY_V4,
	}
	arpChanged, err := applyNeighAction(nlh, nlnh, netlink.FAMILY_V4, action)
	if err != nil {
		return netmanager.EndpointNeighResults{}, errors.Wrapf(err, "could not modify neighbor entry: %+v", nlnh)
	}

	nlnh = &netlink.Neigh{
		IP:           vtepIP,
		HardwareAddr: mac,
		State:        netlink.NUD_PERMANENT,
		LinkIndex:    link.Attrs().Index,
		Family:       unix.AF_BRIDGE,
		Flags:        netlink.NTF_SELF,
	}
	fdbChanged, err := applyNeighAction(nlh, nlnh, unix.AF_BRIDGE, action)
	if err != nil {
		return netmanager.EndpointNeighResults{}, errors.Wrapf(err, "could not modify neighbor entry: %+v", nlnh)
	}

	switch action {
	case neighActionSet:
		return netmanager.EndpointNeighResults{AddedARPEntry: arpChanged, AddedFDBEntry: fdbChanged}, nil
	case neighActionDel:
		return netmanager.EndpointNeighResults{RemovedARPEntry: arpChanged, RemovedFDBEntry: fdbChanged}, nil
	default:
		return netmanager.EndpointNeighResults{}, errors.Errorf("unknown neighbor action %q", action)
	}
}

func applyNeighAction(nlh *netlink.Handle, neigh *netlink.Neigh, family int, action neighAction) (bool, error) {
	neighs, err := nlh.NeighList(neigh.LinkIndex, family)
	if err != nil {
		return false, errors.Wrapf(err, "list neighbor entries")
	}

	exists := neighExists(neighs, *neigh)
	if action == neighActionSet && exists {
		return false, nil
	}
	if action == neighActionDel && !exists {
		return false, nil
	}

	switch action {
	case neighActionSet:
		err = nlh.NeighSet(neigh)
	case neighActionDel:
		err = nlh.NeighDel(neigh)
	default:
		return false, errors.Errorf("unknown neighbor action %q", action)
	}
	if err != nil {
		return false, err
	}

	return true, nil
}

func reconcileNeighs(nlh *netlink.Handle, existingEntries []netlink.Neigh, expectedEntries []netlink.Neigh, isManaged func(netlink.Neigh) bool) (int, int, error) {
	existing := make(map[string]netlink.Neigh, len(existingEntries))
	for _, entry := range existingEntries {
		if isManaged(entry) {
			existing[neighKey(entry)] = entry
		}
	}

	expected := make(map[string]netlink.Neigh, len(expectedEntries))
	for _, entry := range expectedEntries {
		expected[neighKey(entry)] = entry
	}

	added := 0
	for key, entry := range expected {
		if _, ok := existing[key]; ok {
			continue
		}
		err := nlh.NeighSet(&entry)
		if err != nil {
			return added, 0, err
		}
		added++
	}

	removed := 0
	for key, entry := range existing {
		if _, ok := expected[key]; ok {
			continue
		}
		err := nlh.NeighDel(&entry)
		if err != nil {
			return added, removed, err
		}
		removed++
	}

	return added, removed, nil
}

func isManagedARPEntry(neigh netlink.Neigh) bool {
	return neigh.Family == netlink.FAMILY_V4 && neigh.State == netlink.NUD_PERMANENT
}

func isManagedFDBEntry(neigh netlink.Neigh) bool {
	return neigh.Family == unix.AF_BRIDGE && neigh.State == netlink.NUD_PERMANENT && neigh.Flags == netlink.NTF_SELF
}

func neighKey(neigh netlink.Neigh) string {
	return fmt.Sprintf("%d|%d|%d|%d|%s|%s", neigh.LinkIndex, neigh.Family, neigh.State, neigh.Flags, neigh.IP.String(), neigh.HardwareAddr.String())
}

func neighExists(neighs []netlink.Neigh, expected netlink.Neigh) bool {
	for _, neigh := range neighs {
		if neigh.LinkIndex == expected.LinkIndex &&
			neigh.IP.Equal(expected.IP) &&
			neigh.HardwareAddr.String() == expected.HardwareAddr.String() &&
			neigh.State == expected.State &&
			neigh.Family == expected.Family &&
			neigh.Flags == expected.Flags {
			return true
		}
	}

	return false
}
