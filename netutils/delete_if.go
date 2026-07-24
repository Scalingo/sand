package netutils

import (
	"context"

	"github.com/vishvananda/netlink"
	"github.com/vishvananda/netns"
	"golang.org/x/sys/unix"

	"github.com/Scalingo/go-utils/errors/v3"
)

func DeleteInterfaceIfExists(ctx context.Context, nsfd netns.NsHandle, ifname string) error {
	nlh, err := netlink.NewHandleAt(nsfd, unix.NETLINK_ROUTE)
	if err != nil {
		return errors.Wrapf(ctx, err, "get netlink handler of netns")
	}
	defer nlh.Delete()

	link, err := nlh.LinkByName(ifname)
	if _, ok := err.(netlink.LinkNotFoundError); ok {
		return nil
	}
	if err != nil {
		return errors.Wrapf(ctx, err, "get veth interface in container %v", ifname)
	}

	err = nlh.LinkSetDown(link)
	if err != nil {
		return errors.Wrapf(ctx, err, "shutdown link %v", ifname)
	}

	err = nlh.LinkDel(link)
	if err != nil {
		return errors.Wrapf(ctx, err, "remove link %v", ifname)
	}

	return nil
}
