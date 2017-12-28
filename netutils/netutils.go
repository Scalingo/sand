package netutils

import (
	"strings"

	"github.com/Scalingo/sand/netlink"
	"github.com/docker/libnetwork/netutils"
	"github.com/docker/libnetwork/types"
	nl "github.com/vishvananda/netlink"
)

// GenerateIfaceName returns an interface name using the passed in
// prefix and the length of random bytes. The api ensures that the
// there are is no interface which exists with that name.
// From "github.com/docker/libnetwork/netutils"
func GenerateIfaceName(nlh netlink.Handler, prefix string, len int) (string, error) {
	for i := 0; i < 3; i++ {
		name, err := netutils.GenerateRandomName(prefix, len)
		if err != nil {
			continue
		}
		_, err = nlh.LinkByName(name)
		if err != nil {
			if strings.Contains(err.Error(), "not found") {
				return name, nil
			}
			return "", err
		}
	}
	return "", types.InternalErrorf("could not generate interface name")
}

// ParseAddr parses the string representation of an address in the
// form $ip/$netmask $label. The label portion is optional
// From "github.com/vishvananda/netlink"
func ParseAddr(s string) (*netlink.Addr, error) {
	label := ""
	parts := strings.Split(s, " ")
	if len(parts) > 1 {
		s = parts[0]
		label = parts[1]
	}
	m, err := nl.ParseIPNet(s)
	if err != nil {
		return nil, err
	}
	return &netlink.Addr{IPNet: m, Label: label}, nil
}
