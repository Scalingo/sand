package overlay

import (
	"context"

	"github.com/Scalingo/sand/api/types"
	"github.com/Scalingo/sand/netutils"
	"github.com/pkg/errors"
	"github.com/vishvananda/netns"
)

func (m manager) DeleteEndpoint(ctx context.Context, n types.Network, e types.Endpoint) error {
	overlaynsfd, err := netns.GetFromPath(n.NSHandlePath)
	if err != nil {
		return errors.Wrapf(err, "fail to get namespace handler")
	}
	defer overlaynsfd.Close()

	err = netutils.DeleteInterfaceIfExists(ctx, overlaynsfd, e.OverlayVethName)
	if err != nil {
		return errors.Wrapf(err, "fail to delete interface on targetns")
	}

	hostfd, err := netns.Get()
	if err != nil {
		return errors.Wrapf(err, "fail to get host namespace handler")
	}
	defer hostfd.Close()

	err = netutils.DeleteInterfaceIfExists(ctx, hostfd, e.TargetVethName)
	if err != nil {
		return errors.Wrapf(err, "fail to delete interface on host")
	}

	targetfd, err := netns.GetFromPath(e.TargetNetnsPath)
	if err != nil {
		return errors.Wrapf(err, "fail to get host namespace handler")
	}
	defer targetfd.Close()

	err = netutils.DeleteInterfaceIfExists(ctx, targetfd, e.TargetVethName)
	if err != nil {
		return errors.Wrapf(err, "fail to delete interface on targetns")
	}

	return nil
}
