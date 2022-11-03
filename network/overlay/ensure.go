package overlay

import (
	"context"
	"fmt"
	"math/rand"
	"time"

	"github.com/pkg/errors"
	"github.com/vishvananda/netlink"
	"github.com/vishvananda/netlink/nl"
	"github.com/vishvananda/netns"
	"golang.org/x/sys/unix"

	"github.com/Scalingo/sand/api/types"
	"github.com/Scalingo/sand/netnsbuilder"
)

const (
	BridgeName            = "br0"
	VxLANInNSName         = "vxlan0"
	VxLANInHostPrefix     = "vxlan-"
	NetlinkSocketsTimeout = 3 * time.Second
)

func (netm manager) Ensure(ctx context.Context, network types.Network) error {
	m := netnsbuilder.NewManager(netm.config)
	err := m.Create(ctx, network.Name, network)
	if err != nil && err != netnsbuilder.ErrAlreadyExist {
		return errors.Wrapf(err, "fail to create network namspace")
	}

	// Get a netlink handle to the root network namespace
	rootNetlinkHandle, err := netlink.NewHandle(unix.NETLINK_ROUTE)
	if err != nil {
		return errors.Wrapf(err, "could not create netlink handle on initial root namespace")
	}
	defer rootNetlinkHandle.Delete()

	err = rootNetlinkHandle.SetSocketTimeout(NetlinkSocketsTimeout)
	if err != nil {
		return errors.Wrapf(err, "fail to configure timeout on netlink socket")
	}

	// Get namespace file descriptor and netlink handle for VxLAN Namespace to
	// check how it's configured
	nsfd, err := netns.GetFromPath(network.NSHandlePath)
	if err != nil {
		return errors.Wrapf(err, "fail to get namespace handler")
	}
	defer nsfd.Close()

	nlh, err := netlink.NewHandleAt(nsfd, unix.NETLINK_ROUTE)
	if err != nil {
		return errors.Wrapf(err, "fail to get netlink handler of netns")
	}
	defer nlh.Delete()
	err = nlh.SetSocketTimeout(NetlinkSocketsTimeout)
	if err != nil {
		return errors.Wrapf(err, "fail to configure timeout on netlink socket")
	}

	// List all interfaces in the VxLAN namespace
	var link netlink.Link
	links, err := nlh.LinkList()
	if err != nil {
		return errors.Wrapf(err, "fail to list links")
	}

	// There should be a bridge linking all interfaces sharing the same VxLAN
	// network on the host
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

	// Check the gateway IP address is correctly set on the bridge
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

	// Check that the VxLAN interface exist in its dedicated namespace (alongside the bridge)
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

		// Create a VxLAN interface in the root namespace (only way to ensure the
		// kernel does take it into account, creating one in a sub-namespace
		// doesn't work)
		err := rootNetlinkHandle.LinkAdd(vxlan)
		if err != nil {
			return errors.Wrapf(err, "error creating %s interface (VNI: %v)", vxlan.Attrs().Name, network.VxLANVNI)
		}

		link, err := rootNetlinkHandle.LinkByName(vxlan.Attrs().Name)
		if err != nil {
			return errors.Wrapf(err, "fail to get %s link", vxlan.Attrs().Name)
		}

		// Move the VxLAN link in the right namespace to prevent processes in root
		// namespace to use it
		err = rootNetlinkHandle.LinkSetNsFd(link, int(nsfd))
		if err != nil {
			return errors.Wrap(err, "fail to set netns of vxlan")
		}

		err = nlh.LinkSetName(link, VxLANInNSName)
		if err != nil {
			return errors.Wrapf(err, "fail to rename %s to %s in ns", link.Attrs().Name, VxLANInNSName)
		}
	}

	// Plug the VxLAN interface in the bridge by setting its master attribute
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

	// Ensure all interface of the VxLAN namespace are up
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
