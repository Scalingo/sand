package netnsbuilder

import (
	"context"
	"os"

	"github.com/pkg/errors"
	"golang.org/x/sys/unix"

	"github.com/Scalingo/go-utils/logger"
)

func UnmountNetworkNamespace(ctx context.Context, path string) error {
	log := logger.Get(ctx).WithField("mount-netns", path)
	log.Info("unmounting")
	err := unix.Unmount(path, unix.MNT_DETACH)
	if err != nil {
		return errors.Wrapf(err, "fail to umount %v", path)
	}
	return os.Remove(path)
}
