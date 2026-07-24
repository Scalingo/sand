package main

import (
	"context"
	"io"
	"net"
	"sync"

	"github.com/urfave/cli/v3"

	"github.com/Scalingo/go-utils/errors/v3"
	"github.com/Scalingo/go-utils/logger"
	"github.com/Scalingo/sand/api/params"
)

func (a *App) NetworkConnect(ctx context.Context, c *cli.Command) error {
	log := logger.Default()

	client, err := a.sandClient(ctx, c)
	if err != nil {
		return errors.Wrap(ctx, err, "create sand client")
	}

	if c.String("network") == "" || c.String("ip") == "" || c.String("port") == "" {
		return errors.New(ctx, "network, ip and port flags are mandatory")
	}

	listener, err := net.Listen("tcp", ":")
	if err != nil {
		return errors.Wrapf(ctx, err, "bind a socket")
	}
	defer listener.Close()

	log.Infof("Waiting connection on %v", listener.Addr())

	for {
		localConn, err := listener.Accept()
		if err != nil {
			return errors.Wrapf(ctx, err, "accept connection")
		}

		go func(localConn net.Conn) {
			conn, err := client.NetworkConnect(ctx, c.String("network"), params.NetworkConnect{
				IP:   c.String("ip"),
				Port: c.String("port"),
			})
			if err != nil {
				log.WithError(err).Errorf("connect to network %v", c.String("network"))
				return
			}

			wg := &sync.WaitGroup{}
			wg.Add(2)

			closed := false
			go func() {
				defer func() {
					closed = true
					wg.Done()
					localConn.Close()
					conn.Close()
				}()
				log.Info("remote connection opened to the SAND network")
				_, err := io.Copy(localConn, conn)
				if err != io.EOF && err != nil && !closed {
					log.WithError(err).Error("copy data from local socket to remote network")
				}
				log.Info("remote connection closed to the SAND network")
			}()

			go func() {
				defer func() {
					closed = true
					wg.Done()
					localConn.Close()
					conn.Close()
				}()

				_, err := io.Copy(conn, localConn)
				if err != io.EOF && err != nil && !closed {
					log.WithError(err).Error("copy data from remote network to local socket")
				}
				log.Infof("local connection on %v closed", listener.Addr())
			}()

			wg.Wait()
		}(localConn)
	}

	// unreachable
}
