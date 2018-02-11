package docker

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/Scalingo/sand/config"
	"github.com/pkg/errors"
)

const (
	DockerPluginName = "sand"
	DockerPluginFile = "sand.json"
	DockerSpecDir    = "/etc/docker/plugins"
)

type PluginSpec struct {
	Name      string
	Addr      string
	TLSConfig struct {
		InsecureSkipVerify bool
		CAFile             string
		CertFile           string
		KeyFile            string
	}
}

func WritePluginSpecsOnDisk(ctx context.Context, c *config.Config) error {
	err := os.MkdirAll(DockerSpecDir, 0755)
	if err != nil {
		return errors.Wrapf(err, "fail to create docker plugins directory")
	}
	fd, err := os.OpenFile(filepath.Join(DockerSpecDir, DockerPluginFile), os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0600)
	if err != nil {
		return errors.Wrapf(err, "fail to open spec file")
	}
	defer fd.Close()

	scheme := "http"
	if c.HttpTLSCA != "" {
		scheme = "https"
	}

	spec := PluginSpec{
		Name: DockerPluginName,
		Addr: fmt.Sprintf("%s://%s:%d", scheme, c.PublicHostname, c.DockerPluginHttpPort),
	}

	if c.HttpTLSCA != "" {
		spec.TLSConfig.CAFile = c.HttpTLSCA
		spec.TLSConfig.CertFile = c.HttpTLSCert
		spec.TLSConfig.KeyFile = c.HttpTLSKey
	}

	err = json.NewEncoder(fd).Encode(&spec)
	if err != nil {
		return errors.Wrapf(err, "fail to encode spec in JSON")
	}
	return nil
}
