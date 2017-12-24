package netnsbuilder

import (
	"context"
	"os"
	"syscall"

	"github.com/Scalingo/go-internal-tools/logger"
	"github.com/docker/docker/pkg/reexec"
	"github.com/sirupsen/logrus"
)

func init() {
	reexec.Register("sc-netns-create", reexecCreateNamespace)
}

func reexecCreateNamespace() {
	if len(os.Args) < 2 {
		logrus.Fatal("no namespace path provided")
	}
	log := logger.Default().WithField("mount-netns", os.Args[1])
	ctx := logger.ToCtx(context.Background(), log)
	if err := mountNetworkNamespace(ctx, "/proc/self/ns/net", os.Args[1]); err != nil {
		logrus.Fatal(err)
	}
}

func mountNetworkNamespace(ctx context.Context, basePath string, lnPath string) error {
	log := logger.Get(ctx)
	log.Info("mounting")
	return syscall.Mount(basePath, lnPath, "bind", syscall.MS_BIND, "")
}
