package netutils

import (
	"context"
	"syscall"

	"github.com/pkg/errors"
	"github.com/vishvananda/netlink"
	"github.com/vishvananda/netns"
)

func DeleteInterfaceIfExists(ctx context.Context, nsfd netns.NsHandle, ifname string) error {
	nlh, err := netlink.NewHandleAt(nsfd, syscall.NETLINK_ROUTE)
	if err != nil {
		return errors.Wrapf(err, "fail to get netlink handler of netns")
	}

	link, err := nlh.LinkByName(ifname)
	if _, ok := err.(netlink.LinkNotFoundError); ok {
		return nil
	}
	if err != nil {
		return errors.Wrapf(err, "fail to get veth interface in container %v", ifname)
	}

	err = nlh.LinkSetDown(link)
	if err != nil {
		return errors.Wrapf(err, "fail to shutdown link %v", ifname)
	}

	err = nlh.LinkDel(link)
	if err != nil {
		return errors.Wrapf(err, "fail to remove link %v", ifname)
	}

	return nil
}
