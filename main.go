package main

import (
	"fmt"
	"net/http"
	"os"
	"path/filepath"

	"github.com/Scalingo/go-handlers"
	"github.com/Scalingo/go-internal-tools/logger"
	"github.com/Scalingo/networking-agent/config"
	"github.com/Scalingo/networking-agent/web"
	"github.com/docker/docker/pkg/reexec"
	"github.com/sirupsen/logrus"
)

func main() {
	log := logger.Default()
	log.SetLevel(logrus.DebugLevel)

	// If reexec to create network namespace
	if filepath.Base(os.Args[0]) != "networking-agent" {
		log.WithField("args", os.Args).Info("reexec")
	}
	ok := reexec.Init()
	if ok {
		log.WithField("args", os.Args).Info("reexec done")
		return
	}

	c, err := config.Build()
	if err != nil {
		log.WithError(err).Error("fail to generate initial config")
		os.Exit(-1)
	}

	err = c.CreateDirectories()
	if err != nil {
		log.WithError(err).Error("fail to create runtime directories")
		os.Exit(-1)
	}

	r := handlers.NewRouter(log)
	r.Use(handlers.ErrorMiddleware)

	r.HandleFunc("/networks", web.NewNetworksController(c).List).Methods("GET")
	r.HandleFunc("/networks", web.NewNetworksController(c).Create).Methods("POST")
	r.HandleFunc("/networks/{name}", web.NewNetworksController(c).Destroy).Methods("DELETE")

	log.WithField("port", c.HttpPort).Info("Listening")
	http.ListenAndServe(fmt.Sprintf(":%d", c.HttpPort), r)
}
