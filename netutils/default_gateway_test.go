package netutils

import (
	"fmt"
	"net"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestAddIntToIP(t *testing.T) {
	cases := []struct {
		SourceIP  string
		Increment uint64
		ResultIP  string
	}{
		{"10.0.0.0", 1, "10.0.0.1"},
		{"10.0.0.4", 1, "10.0.0.5"},
		{"10.0.0.255", 1, "10.0.1.0"},
		{"10.0.255.255", 1, "10.1.0.0"},
		{"10.0.0.0", 10, "10.0.0.10"},
		{"10.0.0.1", 256, "10.0.1.1"},
	}

	for _, c := range cases {
		t.Run(fmt.Sprintf("%s + %d == %s", c.SourceIP, c.Increment, c.ResultIP), func(t *testing.T) {
			ip := net.ParseIP(c.SourceIP)
			result := AddIntToIP(ip, c.Increment)
			assert.Equal(t, c.ResultIP, result.String())
		})
	}
}
