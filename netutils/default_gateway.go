package netutils

import (
	"fmt"
	"net"

	"gopkg.in/errgo.v1"
)

func DefaultGateway(cidr string) (string, error) {
	ip, netip, err := net.ParseCIDR(cidr)
	if err != nil {
		return "", errgo.Notef(err, "invalid CIDR")
	}
	ip = AddIntToIP(ip, 1)
	return ToCIDR(ip, netip.Mask), nil
}

// Adds the ordinal IP to the current array
// 192.168.0.0 + 53 => 192.168.0.53
func AddIntToIP(ip net.IP, ordinal uint64) net.IP {
	ip = ip.To4()
	v := uint64(ip[0])<<24 + uint64(ip[1])<<16 + uint64(ip[2])<<8 + uint64(ip[3])
	v += ordinal
	v3 := byte(v & 0xFF)
	v2 := byte((v >> 8) & 0xFF)
	v1 := byte((v >> 16) & 0xFF)
	v0 := byte((v >> 24) & 0xFF)
	return net.IPv4(v0, v1, v2, v3)
}

func ToCIDR(ip net.IP, mask net.IPMask) string {
	ones, _ := mask.Size()
	return fmt.Sprintf("%s/%d", ip.String(), ones)
}
