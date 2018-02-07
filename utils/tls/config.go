package tls

import (
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"io/ioutil"

	"github.com/pkg/errors"
)

func NewConfig(caPath, certPath, keyPath string, clientAuth bool) (*tls.Config, error) {
	caCertBytes, err := ioutil.ReadFile(caPath)
	if err != nil {
		return nil, errors.Wrapf(err, "fail to read CA cert file %v", caPath)
	}

	cert, err := tls.LoadX509KeyPair(certPath, keyPath)
	if err != nil {
		return nil, errors.Wrapf(err, "fail to load certificate %v", certPath)
	}

	pool := x509.NewCertPool()
	pool.AppendCertsFromPEM(caCertBytes)

	config := &tls.Config{
		Certificates: []tls.Certificate{cert},
		ClientCAs:    pool,
		RootCAs:      pool,
	}

	if clientAuth {
		config.ClientAuth = tls.RequireAndVerifyClientCert
	}

	config.Rand = rand.Reader
	return config, nil
}
