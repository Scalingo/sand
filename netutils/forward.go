package netutils

import (
	"context"
	"fmt"
	"io"
	"net"
	"runtime"
	"sync"

	"github.com/Scalingo/go-utils/logger"
	"github.com/pkg/errors"
	"github.com/vishvananda/netns"
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

	dstHost := fmt.Sprintf("%s:%s", ip, port)
	dstSocket, err := net.Dial("tcp", dstHost)
	if err != nil {
		return errors.Wrapf(err, "fail to open connection to %v", dstHost)
	}

	wg := &sync.WaitGroup{}
	wg.Add(2)
	go func() {
		defer wg.Done()
		defer dstSocket.Close()
		io.Copy(dstSocket, srcSocket)
		log.Info("end of connection from unix src to dst socket")
	}()

	go func() {
		defer wg.Done()
		defer srcSocket.Close()
		io.Copy(srcSocket, dstSocket)
		log.Info("end of connection from dst socket to src socket")
	}()
	wg.Wait()

	err = netns.Set(current)
	if err != nil {
		return errors.Wrapf(err, "fail to get back to original ns")
	}

	return nil
}
