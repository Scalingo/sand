package graceful

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/Scalingo/go-utils/logger"
)

// setupSignals is catching INT/TERM signals to handle a graceful shutdown operation
// and HUP for a graceful restart. In the case of a restart, the socket is given to the
// child process to keep receiving new connecions while waiting for the old one to finish
// properly.
func (s *service) setupSignals(ctx context.Context) {
	log := logger.Get(ctx)
	ch := make(chan os.Signal, 10)
	signal.Notify(ch, syscall.SIGINT, syscall.SIGTERM, syscall.SIGHUP)
	for {
		sig := <-ch
		switch sig {
		case syscall.SIGINT, syscall.SIGTERM:
			log.Info("request graceful shutdown")
			signal.Stop(ch)
			ctx, cancel := context.WithTimeout(ctx, s.waitDuration)
			defer cancel()
			err := s.shutdown(ctx)
			if err != nil {
				s.stopped <- fmt.Errorf("fail to shutdown service: %v", err)
			}
			close(s.stopped)
			return
		case syscall.SIGHUP:
			log.Info("request graceful restart")
			_, err := s.graceful.StartProcess()
			if err != nil {
				log.WithError(err).Error("fail to start new process")
			}
			ctx, cancel := context.WithTimeout(ctx, s.reloadWaitDuration)
			defer cancel()
			err = s.shutdown(ctx)
			if err != nil {
				s.stopped <- fmt.Errorf("fail to shutdown service: %v", err)
			} else {
				close(s.stopped)
				return
			}
		}
	}
}
