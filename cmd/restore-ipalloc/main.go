package main

import (
	"context"
	"os"

	"github.com/sirupsen/logrus"

	"github.com/Scalingo/go-etcd-lock/v5/lock"
	"github.com/Scalingo/go-utils/logger"
	"github.com/Scalingo/go-utils/logger/plugins/rollbarplugin"
	"github.com/Scalingo/sand/config"
	"github.com/Scalingo/sand/etcd"
	"github.com/Scalingo/sand/ipallocator"
	"github.com/Scalingo/sand/store"
)

func main() {
	rollbarplugin.Register()
	log := logrus.FieldLogger(logger.Default())
	ctx := logger.ToCtx(context.Background(), log)

	c, err := config.Build()
	if err != nil {
		log.WithError(err).Error("fail to generate initial config")
		os.Exit(-1)
	}

	dataStore := store.New(c)

	etcdClient, err := etcd.NewClient()
	if err != nil {
		log.WithError(err).Error("fail to initialize etcd client")
		os.Exit(-1)
	}

	locker := lock.NewEtcdLocker(etcdClient)
	ipAllocator := ipallocator.New(c, dataStore, locker)

	networkID := os.Args[1]
	ip := os.Args[2]

	_, err = ipAllocator.AllocateIP(ctx, networkID, ipallocator.AllocateIPOpts{
		AddressRange: "192.168.100.0/24",
		Address:      ip,
	})
	if err != nil {
		log.WithError(err).Error("fail to allocate ip")
	}
	log.Info("done")
}
