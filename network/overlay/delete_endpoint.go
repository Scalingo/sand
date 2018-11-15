package overlay

import (
	"context"
	"os"

	"github.com/Scalingo/go-internal-tools/logger"
	"github.com/Scalingo/sand/api/types"
	"github.com/Scalingo/sand/netutils"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"github.com/vishvananda/netns"
)

func (m manager) DeleteEndpoint(ctx context.Context, n types.Network, e types.Endpoint) error {
	log := logger.Get(ctx)

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
		return errors.Wrapf(err, "fail to get current thread network namespace")
	}
	defer hostfd.Close()

	err = netutils.DeleteInterfaceIfExists(ctx, hostfd, e.TargetVethName)
	if err != nil {
		return errors.Wrapf(err, "fail to delete interface on host")
	}

	targetfd, err := netns.GetFromPath(e.TargetNetnsPath)
	if err != nil {
		err := errors.Wrapf(err, "fail to get host namespace handle from path")
		if os.IsNotExist(errors.Cause(err)) {
			return err
		}
		log.WithFields(logrus.Fields{
			"error":               err,
			"endpoint_netns_path": e.TargetNetnsPath,
		}).Info("ignore error")
	}
	defer targetfd.Close()

	err = netutils.DeleteInterfaceIfExists(ctx, targetfd, e.TargetVethName)
	if err != nil {
		return errors.Wrapf(err, "fail to delete interface on targetns")
	}

	return nil
}
