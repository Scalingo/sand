package tls

import (
	"context"
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"os"

	"github.com/Scalingo/go-utils/errors/v3"
)

// NewConfig generate a *tls.Config, clientAuth should be false to generate client configuration
func NewConfig(ctx context.Context, caPath, certPath, keyPath string, clientAuth bool) (*tls.Config, error) {
	caCertBytes, err := os.ReadFile(caPath)
	if err != nil {
		return nil, errors.Wrapf(ctx, err, "read CA cert file %v", caPath)
	}

	cert, err := tls.LoadX509KeyPair(certPath, keyPath)
	if err != nil {
		return nil, errors.Wrapf(ctx, err, "load certificate %v", certPath)
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
