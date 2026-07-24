package main

import (
	"context"
	"fmt"
	"time"

	"github.com/urfave/cli/v3"

	"github.com/Scalingo/go-utils/errors/v3"
)

func (a *App) Version(ctx context.Context, c *cli.Command) error {
	fmt.Printf("Client version: %v\n", a.config.Version)

	client, err := a.sandClient(ctx, c)
	if err != nil {
		return errors.Wrap(ctx, err, "create sand client")
	}

	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	version, err := client.Version(ctx)
	if err != nil {
		return errors.Wrapf(ctx, err, "get server version")
	}

	fmt.Printf("Server version: %v\n", version)

	return nil
}
