package main

import (
	"context"
	"fmt"
	"time"

	"github.com/pkg/errors"
	"github.com/urfave/cli"
)

func (a *App) Version(c *cli.Context) error {
	fmt.Printf("Client version: %v\n", a.config.Version)

	client, err := a.sandClient(c)
	if err != nil {
		return err
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	version, err := client.Version(ctx)
	if err != nil {
		return errors.Wrapf(err, "fail to get server version")
	}

	fmt.Printf("Server version: %v\n", version)

	return nil
}
