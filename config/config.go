package config

import (
	"os"

	etcdutils "github.com/Scalingo/go-utils/etcd"

	"github.com/kelseyhightower/envconfig"
	"github.com/pkg/errors"
)

var Version = "v1.0.3"

type Config struct {
	RollbarToken string
	GoEnv        string `default:"development"`
	Version      string `ignore:"true"`
	NetnsPrefix  string `default:"sc-ns-"`
	NetnsPath    string `default:"/var/run/netns"`
	HTTPPort     int    `envconfig:"PORT" default:"9999"`

	// Deprecated: use PeerHostname
	PublicHostname string `envconfig:"PUBLIC_HOSTNAME"`
	// Deprecated: use PeerIP
	PublicIP string `envconfig:"PUBLIC_IP"`

	// PeerHostname and PeerIP are the hostname and IP address of the current node
	// in the network. It is used to build the overlay network and communicate
	// with other nodes in the network.
	//
	// Use Getter GetPeerHostname() and GetPeerIP() for retrocompat with PublicHostname and PublicIP
	PeerHostname string `envconfig:"PEER_HOSTNAME"`
	PeerIP       string `envconfig:"PEER_IP"`

	// APIHostname is the hostname which should be used to contact a SAND endpoint
	// to communicate with its API
	APIHostname string `envconfig:"API_HOSTNAME"`

	EtcdPrefix    string `default:"/sc-net"`
	EtcdHosts     string `envconfig:"ETCD_HOSTS" default:"http://127.0.0.1:2379"`
	EtcdTLSCACert string `envconfig:"ETCD_CACERT"`
	EtcdTLSKey    string `envconfig:"ETCD_TLS_KEY"`
	EtcdTLSCert   string `envconfig:"ETCD_TLS_CERT"`

	HTTPTLSCert string `envconfig:"HTTP_TLS_CERT"`
	HTTPTLSKey  string `envconfig:"HTTP_TLS_KEY"`
	HTTPTLSCA   string `envconfig:"HTTP_TLS_CA"`

	EnableDockerPlugin   bool `envconfig:"ENABLE_DOCKER_PLUGIN"`
	DockerPluginHttpPort int  `default:"9998"`

	MaxVNI int `envconfig:"MAX_VNI" default:"999_999"`
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
	return c.HTTPTLSCA != "" && c.HTTPTLSCert != "" && c.HTTPTLSKey != ""
}

func (c *Config) GetPeerHostname() string {
	if c.PeerHostname == "" {
		return c.PublicHostname
	}
	return c.PeerHostname
}

func (c *Config) GetPeerIP() string {
	if c.PeerIP == "" {
		return c.PublicIP
	}
	return c.PeerIP
}
