package overlay

import (
	"context"
	"fmt"
	"math/rand"
	"syscall"
	"time"

	"github.com/Scalingo/sand/api/types"
	"github.com/Scalingo/sand/netnsbuilder"
	"github.com/docker/libnetwork/ns"
	"github.com/pkg/errors"
	"github.com/vishvananda/netlink"
	"github.com/vishvananda/netlink/nl"
	"github.com/vishvananda/netns"
)

const (
	BridgeName        = "br0"
	VxLANInNSName     = "vxlan0"
	VxLANInHostPrefix = "vxlan-"
)

func (netm manager) Ensure(ctx context.Context, network types.Network) error {
	m := netnsbuilder.NewManager(netm.config)
	err := m.Create(ctx, network.Name, network)
	if err != nil && err != netnsbuilder.ErrAlreadyExist {
		return errors.Wrapf(err, "fail to create network namspace")
	}

	nsfd, err := netns.GetFromPath(network.NSHandlePath)
	if err != nil {
		return errors.Wrapf(err, "fail to get namespace handler")
	}
	defer nsfd.Close()

	nlh, err := netlink.NewHandleAt(nsfd, syscall.NETLINK_ROUTE)
	if err != nil {
		return errors.Wrapf(err, "fail to get netlink handler of netns")
	}
	defer nlh.Delete()

	var link netlink.Link

	links, err := nlh.LinkList()
	if err != nil {
		return errors.Wrapf(err, "fail to list links")
	}

	exist := false
	var bridge *netlink.Bridge
	for _, l := range links {
		if l.Attrs().Name == BridgeName {
			link = l
			bridge = l.(*netlink.Bridge)
			exist = true
			break
		}
	}

	if !exist {
		b := &netlink.Bridge{
			LinkAttrs: netlink.LinkAttrs{
				Name: BridgeName,
			},
		}

		if err := nlh.LinkAdd(b); err != nil {
			return errors.Wrapf(err, "fail to create bridge in namespace")
		}

		link, err = nlh.LinkByName(BridgeName)
		if err != nil {
			return errors.Wrapf(err, "fail to get bridge link")
		}

		bridge = link.(*netlink.Bridge)
	}

	addresses, err := nlh.AddrList(link, nl.FAMILY_V4)
	if err != nil {
		return errors.Wrapf(err, "fail to list addresses of %s", BridgeName)
	}

	if len(addresses) == 0 {
		brAddr, err := netlink.ParseAddr(network.Gateway)
		if err != nil {
			return errors.Wrapf(err, "fail to parse %s IP address", network.Gateway)
		}
		err = nlh.AddrAdd(link, brAddr)
		if err != nil {
			return errors.Wrapf(err, "fail to add %s on bridge", network.Gateway)
		}
	}

	exist = false
	for _, link := range links {
		if link.Attrs().Name == VxLANInNSName {
			exist = true
			break
		}
	}

	if !exist {
		vxlan := &netlink.Vxlan{
			LinkAttrs: netlink.LinkAttrs{Name: fmt.Sprintf("%s%05d", VxLANInHostPrefix, genVxLANSuffix()), MTU: 1450},
			VxlanId:   network.VxLANVNI,
			Learning:  true,
			Port:      4789,
			Proxy:     true,
			L3miss:    true,
			L2miss:    true,
		}

		err := ns.NlHandle().LinkAdd(vxlan)
		if err != nil {
			return errors.Wrapf(err, "error creating %s interface (VNI: %v)", vxlan.Attrs().Name, network.VxLANVNI)
		}

		link, err := ns.NlHandle().LinkByName(vxlan.Attrs().Name)
		if err != nil {
			return errors.Wrapf(err, "fail to get %s link", vxlan.Attrs().Name)
		}

		err = ns.NlHandle().LinkSetNsFd(link, int(nsfd))
		if err != nil {
			return errors.Wrap(err, "fail to set netns of vxlan")
		}

		err = nlh.LinkSetName(link, VxLANInNSName)
		if err != nil {
			return errors.Wrapf(err, "fail to rename %s to %s in ns", link.Attrs().Name, VxLANInNSName)
		}
	}

	link, err = nlh.LinkByName(VxLANInNSName)
	if err != nil {
		return errors.Wrapf(err, "fail to get %s link", VxLANInNSName)
	}

	if link.Attrs().MasterIndex == 0 {
		err := nlh.LinkSetMaster(link, bridge)
		if err != nil {
			return errors.Wrapf(err, "fail to set %s in bridge %s", VxLANInNSName, BridgeName)
		}
	}

	for _, ifName := range []string{"lo", BridgeName, VxLANInNSName} {
		link, err = nlh.LinkByName(ifName)
		if err != nil {
			return errors.Wrapf(err, "fail to get %s link", ifName)
		}
		err = nlh.LinkSetUp(link)
		if err != nil {
			return errors.Wrapf(err, "fail to set %s up", ifName)
		}
	}
	return nil
}

func genVxLANSuffix() uint32 {
	rand.Seed(time.Now().UnixNano())
	return rand.Uint32() % 100000
}
