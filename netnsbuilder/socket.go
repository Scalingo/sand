package netnsbuilder

import (
	"fmt"
	"io"
	"net"
	"os"
	"runtime"
	"sync"

	"github.com/Scalingo/go-utils/logger"
	"github.com/sirupsen/logrus"
	"github.com/vishvananda/netns"
)

func pipeSocket() {
	if len(os.Args) != 5 {
		logrus.Fatalf("%s <ns handle> <ip> <port> <socket unix path>", os.Args[0])
	}
	log := logger.Default().WithFields(logrus.Fields{"ns": os.Args[1], "ip": os.Args[2], "port": os.Args[3], "sock": os.Args[4]})
	log.Info("piping socket")

	runtime.LockOSThread()
	defer runtime.UnlockOSThread()

	conn, err := net.Dial("unix", os.Args[4])
	if err != nil {
		log.WithError(err).Error("fail to open unix connection")
		os.Exit(1)
	}
	unixConn := conn.(*net.UnixConn)

	nsfd, err := netns.GetFromPath(os.Args[1])
	if err != nil {
		log.WithError(err).Error("fail to get network namespace handler")
		os.Exit(-1)
	}

	err = netns.Set(nsfd)
	if err != nil {
		log.WithError(err).Error("fail to set namespace")
		os.Exit(-1)
	}

	socket, err := net.Dial("tcp", fmt.Sprintf("%s:%s", os.Args[2], os.Args[3]))
	if err != nil {
		unixConn.Close()
		log.WithError(err).Error("Fail to open TCP connection")
		os.Exit(-1)
	}
	tcpSocket := socket.(*net.TCPConn)

	wg := &sync.WaitGroup{}
	wg.Add(2)
	go func() {
		defer wg.Done()
		defer unixConn.Close()
		defer tcpSocket.CloseWrite()
		_, err := io.Copy(tcpSocket, unixConn)
		if err != nil && err != io.EOF {
			log.WithError(err).Error("fail to copy stdin to socket")
		}
		log.Info("connection from file descriptor closed")
	}()

	go func() {
		defer wg.Done()
		defer tcpSocket.Close()
		defer unixConn.CloseWrite()
		_, err := io.Copy(unixConn, tcpSocket)
		if err != nil && err != io.EOF {
			log.WithError(err).Error("fail to copy socket to stdout")
		}
		log.Info("connection from ns socket closed")
	}()
	wg.Wait()
}
