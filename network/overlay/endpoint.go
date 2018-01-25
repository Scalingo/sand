package overlay

import (
	"context"
	"net"
	"strings"
	"syscall"

	"github.com/Scalingo/go-internal-tools/logger"
	"github.com/Scalingo/sand/api/types"
	"github.com/Scalingo/sand/netlink"
	"github.com/Scalingo/sand/netutils"
	"github.com/pkg/errors"
	"github.com/vishvananda/netlink/nl"
	"github.com/vishvananda/netns"
)

type overlayEndpoint struct {
	endpoint    types.Endpoint
	targetnsfd  netns.NsHandle
	overlaynsfd netns.NsHandle
	targetnlh   netlink.Handler
	overlaynlh  netlink.Handler
}

func EnsureEndpoint(ctx context.Context, network types.Network, endpoint types.Endpoint) (types.Endpoint, error) {
	targetnsfd, err := netns.GetFromPath(endpoint.TargetNetnsPath)
	if err != nil {
		return endpoint, errors.Wrapf(err, "fail to get target namespace handler: %s", endpoint.TargetNetnsPath)
	}
	defer targetnsfd.Close()

	targetnlh, err := netlink.NewHandleAt(targetnsfd, syscall.NETLINK_ROUTE)
	if err != nil {
		return endpoint, errors.Wrapf(err, "fail to get target namespace netlink handler")
	}

	overlaynsfd, err := netns.GetFromPath(network.NSHandlePath)
	if err != nil {
		return endpoint, errors.Wrapf(err, "fail to get namespace handler %s", network.NSHandlePath)
	}
	defer overlaynsfd.Close()

	overlaynlh, err := netlink.NewHandleAt(overlaynsfd, syscall.NETLINK_ROUTE)
	if err != nil {
		return endpoint, errors.Wrapf(err, "fail to get netlink handler of netns")
	}

	overlayEndpoint := overlayEndpoint{
		endpoint:    endpoint,
		targetnsfd:  targetnsfd,
		overlaynsfd: overlaynsfd,
		targetnlh:   targetnlh,
		overlaynlh:  overlaynlh,
	}

	vethOverlay, vethTarget, err := overlayEndpoint.ensureVethPair(ctx)
	if err != nil {
		return endpoint, errors.Wrapf(err, "fail to create veth pair")
	}

	bridgeLink, err := overlaynlh.LinkByName("br0")
	if err != nil {
		return endpoint, errors.Wrapf(err, "fail to find br0")
	}
	bridge := bridgeLink.(*netlink.Bridge)

	err = overlaynlh.LinkSetMTU(vethOverlay, 1450)
	if err != nil {
		return endpoint, errors.Wrapf(err, "fail to set MTU 1450 on %s", vethOverlay.Attrs().Name)
	}

	err = overlaynlh.LinkSetMaster(vethOverlay, bridge)
	if err != nil {
		return endpoint, errors.Wrapf(err, "fail to add %s in bridge", vethOverlay.Attrs().Name)
	}

	err = targetnlh.LinkSetMTU(vethTarget, 1450)
	if err != nil {
		return endpoint, errors.Wrapf(err, "fail to set MTU to 1450 on %s", vethTarget.Attrs().Name)
	}

	err = targetnlh.LinkSetUp(vethTarget)
	if err != nil {
		return endpoint, errors.Wrapf(err, "fail to set link up %s in target", vethTarget.Attrs().Name)
	}

	addr, err := netutils.ParseAddr(endpoint.TargetVethIP)
	if err != nil {
		return endpoint, errors.Wrapf(err, "fail to parse %s IP address", endpoint.TargetVethIP)
	}

	addrs, err := targetnlh.AddrList(vethTarget, nl.FAMILY_V4)
	if err != nil {
		return endpoint, errors.Wrapf(err, "fail to list addresses of target %v", vethTarget.Attrs().Name)
	}

	exist := false
	for _, a := range addrs {
		if a.IP.String() == addr.IP.String() {
			exist = true
			break
		}
	}

	if !exist {
		err = targetnlh.AddrAdd(vethTarget, addr)
		if err != nil {
			return endpoint, errors.Wrapf(err, "fail to add %s on target veth %v", endpoint.TargetVethIP, vethTarget.Attrs().Name)
		}
	}

	err = overlaynlh.LinkSetUp(vethOverlay)
	if err != nil {
		return endpoint, errors.Wrapf(err, "fail to set link up %s in overlay namespace", vethOverlay.Attrs().Name)
	}

	endpoint.OverlayVethName = vethOverlay.Attrs().Name
	endpoint.OverlayVethMAC = vethOverlay.Attrs().HardwareAddr.String()
	endpoint.TargetVethName = vethTarget.Attrs().Name

	return endpoint, nil
}

func (e overlayEndpoint) ensureVethPair(ctx context.Context) (netlink.Link, netlink.Link, error) {
	log := logger.Get(ctx)
	if e.endpoint.OverlayVethName != "" {
		overlayLink, ok, err := e.isOverlayVethPresent(ctx)
		if err != nil {
			return nil, nil, errors.Wrapf(err, "fail to check existing overlay veth presence")
		}
		if ok {
			targetLink, ok, err := e.isTargetVethPresent(ctx)
			if err != nil {
				return nil, nil, errors.Wrapf(err, "fail to check existing target veth presence")
			}
			if ok {
				return overlayLink, targetLink, nil
			} else {
				log.Info("recreate veth pairs as interface not present on target")
				err = e.overlaynlh.LinkDel(overlayLink)
				if err != nil {
					return nil, nil, errors.Wrapf(err, "fail to remove overlay %s", e.endpoint.OverlayVethName)
				}
			}
		}
	}
	return e.createVethPair(ctx)
}

func (e overlayEndpoint) isOverlayVethPresent(ctx context.Context) (netlink.Link, bool, error) {
	link, err := e.overlaynlh.LinkByName(e.endpoint.OverlayVethName)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			return nil, false, nil
		}
		return nil, false, errors.Wrapf(err, "fail to get link %s", e.endpoint.OverlayVethName)
	}
	return link, true, nil
}

func (e overlayEndpoint) isTargetVethPresent(ctx context.Context) (netlink.Link, bool, error) {
	link, err := e.targetnlh.LinkByName(e.endpoint.TargetVethName)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			return nil, false, nil
		}
		return nil, false, errors.Wrapf(err, "fail to get link %s", e.endpoint.TargetVethName)
	}
	return link, true, nil
}

func (e overlayEndpoint) createVethPair(ctx context.Context) (netlink.Link, netlink.Link, error) {
	vethOverlayName, err := netutils.GenerateIfaceName(e.overlaynlh, "veth", 4)
	if err != nil {
		return nil, nil, errors.Wrapf(err, "fail to generate veth-overlay name")
	}
	vethTargetName, err := netutils.GenerateIfaceName(e.overlaynlh, "veth", 4)
	if err != nil {
		return nil, nil, errors.Wrapf(err, "fail to generate veth-target name")
	}

	mac, err := net.ParseMAC(e.endpoint.TargetVethMAC)
	if err != nil {
		return nil, nil, errors.Wrapf(err, "fail to parse target MAC address")
	}

	veth := &netlink.Veth{
		LinkAttrs: netlink.LinkAttrs{Name: vethOverlayName, TxQLen: 0},
		PeerName:  vethTargetName,
	}
	if err := e.overlaynlh.LinkAdd(veth); err != nil {
		return nil, nil, errors.Wrapf(err, "error creating veth pair")
	}

	vethOverlay, err := e.overlaynlh.LinkByName(vethOverlayName)
	if err != nil {
		return nil, nil, errors.Wrapf(err, "failed to find %s", vethOverlayName)
	}

	vethTarget, err := e.overlaynlh.LinkByName(vethTargetName)
	if err != nil {
		return nil, nil, errors.Wrapf(err, "fail to get %s", vethTargetName)
	}

	err = e.overlaynlh.LinkSetNsFd(vethTarget, int(e.targetnsfd))
	if err != nil {
		return nil, nil, errors.Wrapf(err, "fail to swtch namespace for target veth")
	}

	// Force interface down to avoid 'device busy' error
	err = e.targetnlh.LinkSetDown(vethTarget)
	if err != nil {
		return nil, nil, errors.Wrapf(err, "fail to set down target interface %v", vethTargetName)
	}

	err = e.targetnlh.LinkSetHardwareAddr(vethTarget, mac)
	if err != nil {
		return nil, nil, errors.Wrapf(err, "fail to set mac '%s' on target interface '%v'", mac, vethTargetName)
	}

	inTargetName, err := netutils.GenerateIfaceName(e.targetnlh, "in", 4)
	if err != nil {
		return nil, nil, errors.Wrapf(err, "fail to generate in### name for target interface")
	}

	err = e.targetnlh.LinkSetName(vethTarget, inTargetName)
	if err != nil {
		return nil, nil, errors.Wrapf(err, "fail to rename %s to %s", vethTargetName, inTargetName)
	}

	vethTarget, err = e.targetnlh.LinkByName(inTargetName)
	if err != nil {
		return nil, nil, errors.Wrapf(err, "fail to get %s link", inTargetName)
	}

	return vethOverlay, vethTarget, nil
}
