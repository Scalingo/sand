package network

import (
	"context"
	"fmt"
	"os"
	"syscall"

	"github.com/Scalingo/go-internal-tools/logger"
	"github.com/Scalingo/sand/api/types"
	"github.com/Scalingo/sand/netnsbuilder"
	"github.com/Scalingo/sand/store"
	"github.com/pkg/errors"
	"github.com/vishvananda/netlink"
	"github.com/vishvananda/netns"
)

func (c *repository) Delete(ctx context.Context, network types.Network) error {
	err := c.deleteNamespace(ctx, network)
	if err != nil {
		return errors.Wrapf(err, "fail to delete namespace data")
	}
	err = c.deleteFromStore(ctx, network)
	if err != nil {
		return errors.Wrapf(err, "fail to delete network from store")
	}
	return nil
}

func (c *repository) deleteNamespace(ctx context.Context, network types.Network) error {
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

func (c *repository) deleteFromStore(ctx context.Context, network types.Network) error {
	log := logger.Get(ctx)
	log.WithField("host", c.config.PublicHostname).Info("unlinking host")

	err := c.store.Delete(
		ctx,
		fmt.Sprintf("/nodes/%s/networks/%s", c.config.PublicHostname, network.ID),
	)
	if err != nil {
		return errors.Wrapf(err, "fail to delete network-host link %s from store", network)
	}
	err = c.store.Delete(
		ctx,
		fmt.Sprintf("/nodes-networks/%s/%s", network.ID, c.config.PublicHostname),
	)
	if err != nil {
		return errors.Wrapf(err, "fail to delete network-host link %s from store", network)
	}

	var nets []types.Network
	err = c.store.Get(
		ctx,
		fmt.Sprintf("/nodes-networks/%s/", network.ID),
		true,
		&nets,
	)
	if err != nil && err != store.ErrNotFound {
		return errors.Wrapf(err, "fail to get list of link nodes-networks of %s", network)
	}

	if len(nets) == 0 {
		log.Info("no more link to any nodes, deleting")
		err = c.store.Delete(ctx, network.StorageKey())
		if err != nil {
			return errors.Wrapf(err, "fail to delete network %s from store", network)
		}
	}

	return nil
}
