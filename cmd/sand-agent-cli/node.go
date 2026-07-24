package main

import (
	"context"
	"fmt"

	"github.com/urfave/cli/v3"

	"github.com/Scalingo/go-utils/errors/v3"
)

func (a *App) NodeEnsureNetworkEndpoints(ctx context.Context, c *cli.Command) error {
	client, err := a.sandClient(ctx, c)
	if err != nil {
		return errors.Wrap(ctx, err, "create sand client")
	}

	err = client.NodeEnsureNetworkEndpoints(ctx)
	if err != nil {
		return errors.Wrap(ctx, err, "ensure network endpoints")
	}

	fmt.Println("Network endpoints ensure triggered")
	return nil
}
