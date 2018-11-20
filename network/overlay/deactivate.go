package overlay

import (
	"context"
	"os"
	"strings"
	"syscall"

	"github.com/Scalingo/sand/api/types"
	"github.com/pkg/errors"
	"github.com/vishvananda/netlink"
	"github.com/vishvananda/netns"
)

func (netm manager) Deactivate(ctx context.Context, network types.Network) error {
	nsfd, err := netns.GetFromPath(network.NSHandlePath)
	if os.IsNotExist(err) {
		return nil
	}
	if err != nil {
		return errors.Wrapf(err, "fail to get namespace handler")
	}
	defer nsfd.Close()

	nlh, err := netlink.NewHandleAt(nsfd, syscall.NETLINK_ROUTE)
	if err != nil {
		return errors.Wrapf(err, "fail to get netlink handler of netns")
	}
	defer nlh.Delete()

	links, err := nlh.LinkList()
	if err != nil {
		return errors.Wrapf(err, "fail to list interfaces")
	}
	interfacesToClean := []string{"vxlan0", "br0"}

	for _, link := range links {
		if strings.HasPrefix(link.Attrs().Name, "sand") {
			return errors.Errorf("an endpoint interface is still up: %v", link.Attrs().Name)
		}
	}

	for _, link := range links {
		for _, linkToClean := range interfacesToClean {
			if link.Attrs().Name == linkToClean {
				link, err := nlh.LinkByName(linkToClean)
				if _, ok := err.(netlink.LinkNotFoundError); ok {
					continue
				}
				if err != nil {
					return errors.Wrapf(err, "fail to get %s link", linkToClean)
				}
				err = nlh.LinkDel(link)
				if err != nil {
					return errors.Wrapf(err, "fail to delete %s link", linkToClean)
				}
			}
		}
	}

	return nil
}
