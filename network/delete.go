package network

import (
	"context"
	"syscall"

	"github.com/Scalingo/networking-agent/api/types"
	"github.com/Scalingo/networking-agent/netnsbuilder"
	"github.com/pkg/errors"
	"github.com/vishvananda/netlink"
	"github.com/vishvananda/netns"
)

func (c *repository) Delete(ctx context.Context, network types.Network) error {
	nsfd, err := netns.GetFromPath(network.NSHandlePath)
	if err != nil {
		return errors.Wrapf(err, "fail to get namespace handler")
	}
	defer nsfd.Close()

	nlh, err := netlink.NewHandleAt(nsfd, syscall.NETLINK_ROUTE)
	if err != nil {
		return errors.Wrapf(err, "fail to get netlink handler of netns")
	}

	for _, name := range []string{"vxlan0", "br0"} {
		link, err := nlh.LinkByName(name)
		if _, ok := err.(netlink.LinkNotFoundError); ok {
			continue
		}
		if err != nil {
			return errors.Wrapf(err, "fail to get %s link", name)
		}
		err = nlh.LinkDel(link)
		if err != nil {
			return errors.Wrapf(err, "fail to delete %s link", name)
		}
	}

	nlh.Delete()

	err = netnsbuilder.UnmountNetworkNamespace(ctx, network.NSHandlePath)
	if err != nil {
		return errors.Wrapf(err, "fail to umount network namespace netns handle %v", network.NSHandlePath)
	}
	return nil
}
