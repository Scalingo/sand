package netnsbuilder

import (
	"context"
	"os"

	"github.com/moby/moby/pkg/reexec"
	"github.com/sirupsen/logrus"
	"golang.org/x/sys/unix"

	"github.com/Scalingo/go-utils/logger"
)

func init() {
	reexec.Register("sc-netns-create", reexecCreateNamespace)
}

func reexecCreateNamespace() {
	ctx := context.Background()
	if len(os.Args) < 2 {
		logrus.Fatal("no namespace path provided")
	}
	ctx, log := logger.WithFieldToCtx(
		ctx, "mount-netns", os.Args[1],
	)
	if err := mountNetworkNamespace(ctx, "/proc/self/ns/net", os.Args[1]); err != nil {
		log.WithError(err).Fatal("mount network namespace")
	}
}

func mountNetworkNamespace(ctx context.Context, basePath string, lnPath string) error {
	log := logger.Get(ctx)
	log.Info("mounting")
	return unix.Mount(basePath, lnPath, "bind", unix.MS_BIND, "")
}
