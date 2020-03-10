package store

import (
	"strings"

	"github.com/Scalingo/sand/config"
)

func prefixedKey(config *config.Config, key string) string {
	if strings.HasPrefix(key, config.EtcdPrefix) {
		return key
	}
	return config.EtcdPrefix + key
}
