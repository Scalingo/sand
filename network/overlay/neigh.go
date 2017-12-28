package overlay

import (
	"context"
	"fmt"
	"net"
	"syscall"

	"github.com/Scalingo/go-internal-tools/logger"
	"github.com/Scalingo/sand/api/types"
	"github.com/Scalingo/sand/config"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"github.com/vishvananda/netlink"
	"github.com/vishvananda/netns"
)

func EnsureEndpointsNeigh(ctx context.Context, config *config.Config, network types.Network, endpoints []types.Endpoint) error {
	log := logger.Get(ctx)
	for _, endpoint := range endpoints {
		log = log.WithFields(logrus.Fields{
			"endpoint_id":              endpoint.ID,
			"endpoint_target_ip":       endpoint.TargetVethIP,
			"endpoint_target_hostname": endpoint.Hostname,
		})
		ctx = logger.ToCtx(ctx, log)
		err := AddEndpointNeigh(ctx, config, network, endpoint)
		if err != nil {
			return errors.Wrapf(err, "fail to add endpoint ARP/FDB neigh rules")
		}
	}
	return nil
}

func AddEndpointNeigh(ctx context.Context, config *config.Config, network types.Network, endpoint types.Endpoint) error {
	log := logger.Get(ctx)

	// No rule to add for endpoint located on the current server
	if endpoint.HostIP == config.PublicIP {
		return nil
	}

	log.Info("add endpoint ARP/FDB rules")

	nsfd, err := netns.GetFromPath(network.NSHandlePath)
	if err != nil {
		return errors.Wrapf(err, "fail to get namespace handler")
	}
	defer nsfd.Close()

	nlh, err := netlink.NewHandleAt(nsfd, syscall.NETLINK_ROUTE)
	if err != nil {
		return errors.Wrapf(err, "fail to get netlink handler of netns")
	}

	link, err := nlh.LinkByName(VxLANInNSName)
	if err != nil {
		return errors.Wrapf(err, "fail to get vxlan interface")
	}

	ip, _, err := net.ParseCIDR(endpoint.TargetVethIP)
	if err != nil {
		return errors.Wrapf(err, "fail to parse IP of %v '%s'", endpoint.TargetVethName, endpoint.TargetVethIP)
	}
	mac, err := net.ParseMAC(endpoint.TargetVethMAC)
	if err != nil {
		return errors.Wrapf(err, "fail to parse MAC of %v '%s'", endpoint.TargetVethName, endpoint.TargetVethMAC)
	}
	vtepIP := net.ParseIP(endpoint.HostIP)
	if vtepIP == nil {
		return errors.Errorf("fail to parse endpoint host IP (VTEP IP) '%s'", endpoint.HostIP)
	}

	nlnh := &netlink.Neigh{
		IP:           ip,
		HardwareAddr: mac,
		State:        netlink.NUD_PERMANENT,
		LinkIndex:    link.Attrs().Index,
	}
	if err := nlh.NeighSet(nlnh); err != nil {
		return errors.Wrapf(err, "could not add neighbor entry: %+v", nlnh)
	}

	nlnh = &netlink.Neigh{
		IP:           vtepIP,
		HardwareAddr: mac,
		State:        netlink.NUD_PERMANENT,
		LinkIndex:    link.Attrs().Index,
		Family:       syscall.AF_BRIDGE,
		Flags:        netlink.NTF_SELF,
	}
	if err := nlh.NeighSet(nlnh); err != nil {
		return fmt.Errorf("could not add neighbor entry:%+v error:%v", nlnh, err)
	}
	return nil
}
