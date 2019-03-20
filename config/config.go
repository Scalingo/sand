package config

import (
	"os"

	etcdutils "github.com/Scalingo/go-utils/etcd"

	"github.com/kelseyhightower/envconfig"
	"github.com/pkg/errors"
)

var Version = "v0.6-dev"

type Config struct {
	RollbarToken   string
	GoEnv          string `default:"development"`
	Version        string `ignore:"true"`
	NetnsPrefix    string `default:"sc-ns-"`
	NetnsPath      string `default:"/var/run/netns"`
	HttpPort       int    `default:"9999"`
	PublicHostname string `envconfig:"PUBLIC_HOSTNAME"`
	PublicIP       string `envconfig:"PUBLIC_IP"`

	EtcdPrefix    string `default:"/sc-net"`
	EtcdHosts     string `envconfig:"ETCD_HOSTS" default:"http://127.0.0.1:2379"`
	EtcdTLSCACert string `envconfig:"ETCD_CACERT"`
	EtcdTLSKey    string `envconfig:"ETCD_TLS_KEY"`
	EtcdTLSCert   string `envconfig:"ETCD_TLS_CERT"`

	HttpTLSCert string `envconfig:"HTTP_TLS_CERT"`
	HttpTLSKey  string `envconfig:"HTTP_TLS_KEY"`
	HttpTLSCA   string `envconfig:"HTTP_TLS_CA"`

	EnableDockerPlugin   bool `envconfig:"ENABLE_DOCKER_PLUGIN"`
	DockerPluginHttpPort int  `default:"9998"`
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

	c.Version = Version

	return &c, nil
}

func (c *Config) checkEtcdConfig() error {
	_, err := etcdutils.ConfigFromEnv()
	if err != nil {
		return errors.Wrap(err, "fail to get etcd config from environment")
	}

	return nil
}

func (c *Config) CreateDirectories() error {
	return os.MkdirAll(c.NetnsPath, 0700)
}

func (c *Config) IsHttpTLSEnabled() bool {
	return c.HttpTLSCA != "" && c.HttpTLSCert != "" && c.HttpTLSKey != ""
}
