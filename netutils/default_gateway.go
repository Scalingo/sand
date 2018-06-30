package netutils

import (
	"net"

	"gopkg.in/errgo.v1"
)

func DefaultGateway(cidr string) (string, error) {
	ip, _, err := net.ParseCIDR(cidr)
	if err != nil {
		return "", errgo.Notef(err, "invalid CIDR")
	}
	AddIntToIP(ip, 1)
	return ip.String(), nil
}

// Adds the ordinal IP to the current array
// 192.168.0.0 + 53 => 192.168.0.53
func AddIntToIP(array []byte, ordinal uint64) {
	for i := len(array) - 1; i >= 0; i-- {
		array[i] |= (byte)(ordinal & 0xff)
		ordinal >>= 8
	}
}
