package config

import (
	"os"

	"github.com/kelseyhightower/envconfig"
)

type Config struct {
	RollbarToken string
	NetnsPrefix  string `default:"sc-ns-"`
	NetnsPath    string `default:"/var/run/netns"`
	HttpPort     int    `default:"9999"`
}

func Build() *Config {
	var c Config
	envconfig.Process("", &c)
	return &c
}

func (c *Config) CreateDirectories() error {
	return os.MkdirAll(c.NetnsPath, 0700)
}
