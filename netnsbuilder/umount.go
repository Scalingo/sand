package netnsbuilder

import (
	"context"
	"os"
	"syscall"

	"github.com/Scalingo/go-internal-tools/logger"
	"github.com/pkg/errors"
)

func UnmountNetworkNamespace(ctx context.Context, path string) error {
	log := logger.Get(ctx).WithField("mount-netns", path)
	log.Info("unmounting")
	err := syscall.Unmount(path, syscall.MNT_DETACH)
	if err != nil {
		return errors.Wrapf(err, "fail to umount %v", path)
	}
	return os.Remove(path)
}
