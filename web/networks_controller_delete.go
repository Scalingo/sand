package web

import (
	"net/http"
	"syscall"

	"github.com/Scalingo/go-internal-tools/logger"
	"github.com/Scalingo/networking-agent/api/types"
	"github.com/Scalingo/networking-agent/netnsbuilder"
	"github.com/Scalingo/networking-agent/store"
	"github.com/pkg/errors"
	"github.com/vishvananda/netlink"
	"github.com/vishvananda/netns"
)

func (c NetworksController) Destroy(w http.ResponseWriter, r *http.Request, p map[string]string) error {
	w.Header().Set("Content-Type", "application/json")
	ctx := r.Context()
	log := logger.Get(ctx)

	log = log.WithField("ns-name", p["name"])
	ctx = logger.ToCtx(ctx, log)

	network := types.Network{Name: p["name"]}
	err := c.Store.Get(ctx, network.StorageKey(), false, &network)
	if err != nil && err != store.ErrNotFound {
		return errors.Wrapf(err, "fail to get network %s from store", network.Name)
	}
	if err == store.ErrNotFound {
		w.WriteHeader(404)
		return nil
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

	err = c.Store.Delete(ctx, network.StorageKey())
	if err != nil {
		return errors.Wrapf(err, "fail to delete network %s from store", network.Name)
	}

	w.WriteHeader(204)
	return nil
}
