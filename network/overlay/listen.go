package overlay

import (
	"context"

	"github.com/Scalingo/go-utils/errors/v3"
	"github.com/Scalingo/sand/api/types"
)

func (m manager) ListenNetworkChange(ctx context.Context, n types.Network) error {
	_, err := m.listener.Add(ctx, m, n)
	if err != nil {
		return errors.Wrapf(ctx, err, "add network on listener")
	}
	return nil
}

func (m manager) StopListenNetworkChange(ctx context.Context, n types.Network) error {
	err := m.listener.Remove(ctx, n)
	if err != nil {
		return errors.Wrapf(ctx, err, "remove network listener")
	}
	return nil
}
