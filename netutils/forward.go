package netutils

import (
	"context"
	"fmt"
	"io"
	"net"
	"runtime"
	"sync"

	"github.com/pkg/errors"
	"github.com/vishvananda/netns"

	"github.com/Scalingo/go-utils/logger"
)

type Conn interface {
	Write([]byte) (int, error)
	Read([]byte) (int, error)
	Close() error
	CloseWrite() error
}

func ForwardConnection(ctx context.Context, srcSocket net.Conn, ns, ip, port string) error {
	log := logger.Get(ctx)

	runtime.LockOSThread()
	defer runtime.UnlockOSThread()

	dst, err := netns.GetFromPath(ns)
	if err != nil {
		return errors.Wrapf(err, "fail to get dest namespace handler %v", ns)
	}
	defer dst.Close()

	current, err := netns.Get()
	if err != nil {
		return errors.Wrapf(err, "fail to get current namespace handler")
	}
	defer current.Close()

	err = netns.Set(dst)
	if err != nil {
		return errors.Wrapf(err, "fail to set current namespace to dst %v", dst)
	}
	defer func() {
		err = netns.Set(current)
		if err != nil {
			log.WithError(err).Error("fail to get back to original ns")
		}
	}()

	dstHost := fmt.Sprintf("%s:%s", ip, port)
	dialer := net.Dialer{}
	dstSocket, err := dialer.DialContext(ctx, "tcp", dstHost)
	if err != nil {
		return errors.Wrapf(err, "fail to open connection to %v", dstHost)
	}
	fmt.Println("Connected to", dstHost)

	wg := &sync.WaitGroup{}
	wg.Add(2)

	// Error logs are Info as we're only proxying connections, if one is stopped abruptely
	// The real error is either from the client side or the destination side, it's not an
	// error for the proxy itself
	go func() {
		defer wg.Done()
		defer dstSocket.Close()
		_, err := io.Copy(dstSocket, srcSocket)
		if err != io.EOF && err != nil {
			log.WithError(err).Info("end of connection from unix src to dst socket with error")
			return
		}
		log.Debug("end of connection from unix src to dst socket")
	}()

	go func() {
		defer wg.Done()
		defer srcSocket.Close()
		_, err := io.Copy(srcSocket, dstSocket)
		if err != io.EOF && err != nil {
			log.WithError(err).Info("end of connection from dst socket to src socket with error")
			return
		}
		log.Debug("end of connection from dst socket to src socket")
	}()
	wg.Wait()

	return nil
}
