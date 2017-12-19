package config

import (
	"net/url"
	"os"
	"strings"

	"github.com/kelseyhightower/envconfig"
	"github.com/pkg/errors"
)

type Config struct {
	RollbarToken   string
	GoEnv          string `default:"development"`
	NetnsPrefix    string `default:"sc-ns-"`
	NetnsPath      string `default:"/var/run/netns"`
	HttpPort       int    `default:"9999"`
	PublicHostname string

	EtcdPrefix  string `default:"/sc-net"`
	EtcdURL     string `envconfig:"ETCD_URL" default:"http://127.0.0.1:2379"`
	EtcdTLSCert string `envconfig:"ETCD_TLS_CERT"`
	EtcdTLSKey  string `envconfig:"ETCD_TLS_KEY"`
	EtcdTLSCA   string `envconfig:"ETCD_TLS_CA"`

	EtcdWithTLS   bool
	EtcdEndpoints []string
}

func Build() (*Config, error) {
	var c Config
	envconfig.Process("", &c)

	err := c.checkEtcdConfig()
	if err != nil {
		return nil, errors.Wrapf(err, "fail to build etcd config")
	}

	if c.PublicHostname == "" {
		h, err := os.Hostname()
		if err != nil {
			return nil, errors.Wrapf(err, "fail to get hostname")
		}
		c.PublicHostname = h
	}

	return &c, nil
}

func (c *Config) checkEtcdConfig() error {
	url, err := url.Parse(c.EtcdURL)
	if err != nil {
		return errors.Wrapf(err, "not a valid URL: %s", c.EtcdURL)
	}
	c.EtcdEndpoints = strings.Split(url.Host, ",")

	if url.Scheme == "https" {
		c.EtcdWithTLS = true
		_, err := os.Stat(c.EtcdTLSCert)
		if err != nil {
			return errors.Wrap(err, "invalid etcd TLS cert")
		}

		_, err = os.Stat(c.EtcdTLSKey)
		if err != nil {
			return errors.Wrap(err, "invalid etcd TLS cert")
		}

		_, err = os.Stat(c.EtcdTLSCA)
		if err != nil {
			return errors.Wrap(err, "invalid etcd TLS cert")
		}
	}

	return nil
}

func (c *Config) CreateDirectories() error {
	return os.MkdirAll(c.NetnsPath, 0700)
}
