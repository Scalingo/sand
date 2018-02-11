package overlay

import (
	"context"

	"github.com/Scalingo/sand/api/types"
	"github.com/pkg/errors"
)

func (m manager) ListenNetworkChange(ctx context.Context, n types.Network) error {
	_, err := m.listener.Add(ctx, m, n)
	if err != nil {
		return errors.Wrapf(err, "fail to add network on listener")
	}
	return nil
}
