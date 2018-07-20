package sandenter

import (
	"context"
	"net"

	"github.com/Scalingo/sand/client/sand"
	"github.com/pkg/errors"
)

type handle struct {
	client sand.Client
	nid    string
}

func NewHandle(client sand.Client, networkID string) handle {
	return handle{client: client, nid: networkID}
}

func (h handle) Dial(ctx context.Context) func(proto, dest string) (net.Conn, error) {
	return func(proto, dest string) (net.Conn, error) {
		network, err := h.client.NetworksShow(ctx, h.nid)
		if err != nil {
			return nil, errors.Wrapf(err, "fail to get network")
		}

		func
		network.NSHandlePath
		return nil, nil
	}
}
