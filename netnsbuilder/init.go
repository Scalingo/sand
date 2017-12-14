package netnsbuilder

import (
	"os"
	"syscall"

	"github.com/docker/docker/pkg/reexec"
	"github.com/sirupsen/logrus"
)

func init() {
	reexec.Register("netns-create", reexecCreateNamespace)
}

func reexecCreateNamespace() {
	if len(os.Args) < 2 {
		logrus.Fatal("no namespace path provided")
	}
	if err := mountNetworkNamespace("/proc/self/ns/net", os.Args[1]); err != nil {
		logrus.Fatal(err)
	}
}

func mountNetworkNamespace(basePath string, lnPath string) error {
	return syscall.Mount(basePath, lnPath, "bind", syscall.MS_BIND, "")
}
