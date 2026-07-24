package overlay

import (
	"context"
	"os"

	"github.com/vishvananda/netns"

	"github.com/Scalingo/go-utils/errors/v3"
	"github.com/Scalingo/sand/api/types"
	"github.com/Scalingo/sand/netutils"
	"github.com/Scalingo/sand/network/netmanager"
)

func (m manager) DeleteEndpoint(ctx context.Context, n types.Network, e types.Endpoint) error {
	overlaynsfd, err := netns.GetFromPath(n.NSHandlePath)
	if os.IsNotExist(err) {
		return netmanager.EndpointAlreadyDisabledErr
	} else if err != nil {
		return errors.Wrapf(ctx, err, "get namespace handler")
	}
	defer overlaynsfd.Close()

	err = netutils.DeleteInterfaceIfExists(ctx, overlaynsfd, e.OverlayVethName)
	if err != nil {
		return errors.Wrapf(ctx, err, "delete interface on targetns")
	}

	hostfd, err := netns.Get()
	if err != nil {
		return errors.Wrapf(ctx, err, "get current thread network namespace")
	}
	defer hostfd.Close()

	err = netutils.DeleteInterfaceIfExists(ctx, hostfd, e.TargetVethName)
	if err != nil {
		return errors.Wrapf(ctx, err, "delete interface on host")
	}

	targetfd, err := netns.GetFromPath(e.TargetNetnsPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return errors.Wrapf(ctx, err, "get host namespace handle from path")
	}
	defer targetfd.Close()

	err = netutils.DeleteInterfaceIfExists(ctx, targetfd, e.TargetVethName)
	if err != nil {
		return errors.Wrapf(ctx, err, "delete interface on targetns")
	}

	return nil
}
