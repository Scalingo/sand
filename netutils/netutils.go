package netutils

import (
	"crypto/rand"
	"encoding/hex"
	"io"
	"strings"

	nl "github.com/vishvananda/netlink"
	"gopkg.in/errgo.v1"

	"github.com/Scalingo/sand/netlink"
)

// GenerateIfaceName returns an interface name using the passed in
// prefix and the length of random bytes. The api ensures that the
// there are is no interface which exists with that name.
func GenerateIfaceName(nlh netlink.Handler, prefix string, len int) (string, error) {
	for i := 0; i < 3; i++ {
		name, err := GenerateRandomName(prefix, len)
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
	return "", errgo.New("could not generate interface name after 3 attempts")
}

// GenerateRandomName returns a new name joined with a prefix.  This size
// specified is used to truncate the randomly generated value
func GenerateRandomName(prefix string, size int) (string, error) {
	id := make([]byte, 32)
	if _, err := io.ReadFull(rand.Reader, id); err != nil {
		return "", err
	}
	return prefix + hex.EncodeToString(id)[:size], nil
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
