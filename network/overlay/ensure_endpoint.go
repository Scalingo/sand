package overlay

import (
	"context"
	"net"

	"github.com/pkg/errors"
	"github.com/vishvananda/netlink/nl"
	"github.com/vishvananda/netns"
	"golang.org/x/sys/unix"

	"github.com/Scalingo/go-utils/logger"
	"github.com/Scalingo/sand/api/params"
	"github.com/Scalingo/sand/api/types"
	"github.com/Scalingo/sand/netlink"
	"github.com/Scalingo/sand/netutils"
)

type overlayEndpoint struct {
	endpoint    types.Endpoint
	hostnsfd    netns.NsHandle
	targetnsfd  netns.NsHandle
	overlaynsfd netns.NsHandle
	hostnlh     netlink.Handler
	targetnlh   netlink.Handler
	overlaynlh  netlink.Handler
}

func (m manager) EnsureEndpoint(ctx context.Context, network types.Network, endpoint types.Endpoint, params params.EndpointActivate) (types.Endpoint, error) {
	hostnsfd, err := netns.Get()
	if err != nil {
		return endpoint, errors.Wrapf(err, "fail to get current thread network namespace")
	}
	defer hostnsfd.Close()

	var targetnsfd netns.NsHandle
	if !params.MoveVeth {
		targetnsfd = hostnsfd
	} else {
		targetnsfd, err = netns.GetFromPath(endpoint.TargetNetnsPath)
		if err != nil {
			return endpoint, errors.Wrapf(err, "fail to get target namespace handler: %s", endpoint.TargetNetnsPath)
		}
		defer targetnsfd.Close()
	}

	hostnlh, err := netlink.NewHandleAt(hostnsfd, unix.NETLINK_ROUTE)
	if err != nil {
		return endpoint, errors.Wrapf(err, "fail to get host namespace handler")
	}
	defer hostnlh.Delete()

	targetnlh, err := netlink.NewHandleAt(targetnsfd, unix.NETLINK_ROUTE)
	if err != nil {
		return endpoint, errors.Wrapf(err, "fail to get target namespace netlink handler")
	}
	defer targetnlh.Delete()

	overlaynsfd, err := netns.GetFromPath(network.NSHandlePath)
	if err != nil {
		return endpoint, errors.Wrapf(err, "fail to get namespace handler %s", network.NSHandlePath)
	}
	defer overlaynsfd.Close()

	overlaynlh, err := netlink.NewHandleAt(overlaynsfd, unix.NETLINK_ROUTE)
	if err != nil {
		return endpoint, errors.Wrapf(err, "fail to get netlink handler of netns")
	}
	defer overlaynlh.Delete()

	overlayEndpoint := overlayEndpoint{
		endpoint:    endpoint,
		hostnsfd:    hostnsfd,
		targetnsfd:  targetnsfd,
		overlaynsfd: overlaynsfd,
		hostnlh:     hostnlh,
		targetnlh:   targetnlh,
		overlaynlh:  overlaynlh,
	}

	vethOverlay, vethTarget, err := overlayEndpoint.ensureVethPair(ctx, params)
	if err != nil {
		return endpoint, errors.Wrapf(err, "fail to create veth pair")
	}

	bridge, err := overlaynlh.LinkByName("br0")
	if err != nil {
		return endpoint, errors.Wrapf(err, "fail to find br0")
	}

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

	if params.SetAddr {
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

func (e overlayEndpoint) ensureVethPair(ctx context.Context, params params.EndpointActivate) (netlink.Link, netlink.Link, error) {
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
	return e.createVethPair(ctx, params)
}

func (e overlayEndpoint) isOverlayVethPresent(ctx context.Context) (netlink.Link, bool, error) {
	link, err := e.overlaynlh.LinkByName(e.endpoint.OverlayVethName)
	if _, ok := err.(netlink.LinkNotFoundError); ok {
		return nil, false, nil
	} else if err != nil {
		return nil, false, errors.Wrapf(err, "fail to get link %s", e.endpoint.OverlayVethName)
	}
	return link, true, nil
}

// isTargetVethPresent is a little more tricky as docker#libnetwork like to rename interfaces
// So interfaces should be listed anc MAC compared in ordered to know if it is present or not
func (e overlayEndpoint) isTargetVethPresent(ctx context.Context) (netlink.Link, bool, error) {
	links, err := e.targetnlh.LinkList()
	if err != nil {
		return nil, false, errors.Wrapf(err, "fail to list links of target namespace")
	}
	for _, l := range links {
		if l.Attrs().HardwareAddr.String() == e.endpoint.TargetVethMAC {
			return l, true, nil
		}
	}

	return nil, false, nil
}

func (e overlayEndpoint) createVethPair(ctx context.Context, params params.EndpointActivate) (netlink.Link, netlink.Link, error) {
	vethOverlayName, err := netutils.GenerateIfaceName(e.hostnlh, "sand", 4)
	if err != nil {
		return nil, nil, errors.Wrapf(err, "fail to generate veth-overlay name")
	}
	vethTargetName, err := netutils.GenerateIfaceName(e.hostnlh, "sand", 4)
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
	if err := e.hostnlh.LinkAdd(veth); err != nil {
		return nil, nil, errors.Wrapf(err, "error creating veth pair")
	}

	vethOverlay, err := e.hostnlh.LinkByName(vethOverlayName)
	if err != nil {
		return nil, nil, errors.Wrapf(err, "failed to find %s", vethOverlayName)
	}

	err = e.hostnlh.LinkSetNsFd(vethOverlay, int(e.overlaynsfd))
	if err != nil {
		return nil, nil, errors.Wrapf(err, "fail to switch namespace for overlay veth")
	}

	vethTarget, err := e.hostnlh.LinkByName(vethTargetName)
	if err != nil {
		return nil, nil, errors.Wrapf(err, "fail to get %s", vethTargetName)
	}

	// Force interface down to avoid 'device busy' error
	err = e.hostnlh.LinkSetDown(vethTarget)
	if err != nil {
		return nil, nil, errors.Wrapf(err, "fail to set down target interface %v", vethTargetName)
	}

	err = e.hostnlh.LinkSetHardwareAddr(vethTarget, mac)
	if err != nil {
		return nil, nil, errors.Wrapf(err, "fail to set mac '%s' on target interface '%v'", mac, vethTargetName)
	}

	if params.MoveVeth {
		err = e.hostnlh.LinkSetNsFd(vethTarget, int(e.targetnsfd))
		if err != nil {
			return nil, nil, errors.Wrapf(err, "fail to switch namespace for target veth")
		}

		vethTarget, err = e.targetnlh.LinkByName(vethTargetName)
		if err != nil {
			return nil, nil, errors.Wrapf(err, "fail to get target veth in target ns")
		}
	}

	return vethOverlay, vethTarget, nil
}
