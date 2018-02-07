package overlay

import (
	"context"
	"syscall"

	"github.com/Scalingo/sand/api/types"
	"github.com/pkg/errors"
	"github.com/vishvananda/netlink"
	"github.com/vishvananda/netns"
)

func (m manager) DeleteEndpoint(ctx context.Context, n types.Network, e types.Endpoint) error {
	nsfd, err := netns.GetFromPath(e.TargetNetnsPath)
	if err != nil {
		return errors.Wrapf(err, "fail to get namespace handler")
	}
	defer nsfd.Close()

	nlh, err := netlink.NewHandleAt(nsfd, syscall.NETLINK_ROUTE)
	if err != nil {
		return errors.Wrapf(err, "fail to get netlink handler of netns")
	}

	link, err := nlh.LinkByName(e.TargetVethName)
	if err != nil {
		return errors.Wrapf(err, "fail to get veth interface in container %v", e.TargetVethName)
	}

	err = nlh.LinkSetDown(link)
	if err != nil {
		return errors.Wrapf(err, "fail to shutdown link %v", e.TargetVethName)
	}

	err = nlh.LinkDel(link)
	if err != nil {
		return errors.Wrapf(err, "fail to remove link %v", e.TargetVethName)
	}

	return nil
}
