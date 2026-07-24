package netnsbuilder

import (
	"context"
	stderrors "errors"
	"os"
	"os/exec"

	"github.com/moby/moby/pkg/reexec"
	"golang.org/x/sys/unix"

	"github.com/Scalingo/go-utils/errors/v3"

	"github.com/Scalingo/sand/api/types"
	"github.com/Scalingo/sand/config"
)

var (
	ErrAlreadyExist = stderrors.New("network namespace already exists")
)

type Manager interface {
	Create(context.Context, string, types.Network) error
}

type manager struct {
	Config *config.Config
}

func NewManager(c *config.Config) Manager {
	return &manager{Config: c}
}

func (m *manager) Create(ctx context.Context, name string, n types.Network) error {
	_, err := os.Stat(n.NSHandlePath)
	if !os.IsNotExist(err) && err != nil {
		return errors.Wrap(ctx, err, "create namespace")
	} else if err == nil {
		return ErrAlreadyExist
	}

	err = m.createNS(ctx, n.NSHandlePath)
	if err != nil {
		return errors.Wrapf(ctx, err, "create new namespace")
	}

	return nil
}

func (m *manager) createNS(ctx context.Context, path string) error {
	f, err := os.Create(path)
	if err != nil {
		return errors.Wrap(ctx, err, "touch netns mountpoint file")
	}
	err = f.Close()
	if err != nil {
		return errors.Wrap(ctx, err, "close netns mountpoint file")
	}

	cmd := &exec.Cmd{
		Path:        reexec.Self(),
		Args:        append([]string{"sc-netns-create"}, path),
		Stdout:      os.Stdout,
		Stderr:      os.Stderr,
		SysProcAttr: &unix.SysProcAttr{Cloneflags: unix.CLONE_NEWNET},
	}
	err = cmd.Run()
	if err != nil {
		return errors.Wrap(ctx, err, "namespace creation reexec command failed")
	}

	return nil
}
