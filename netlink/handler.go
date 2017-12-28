package netlink

import (
	"net"

	"github.com/vishvananda/netlink"
	"github.com/vishvananda/netns"
)

type Addr = netlink.Addr
type Link = netlink.Link
type Bridge = netlink.Bridge
type Neigh = netlink.Neigh
type Veth = netlink.Veth
type LinkAttrs = netlink.LinkAttrs

type Handler interface {
	AddrList(Link, int) ([]netlink.Addr, error)
	LinkList() ([]Link, error)
	LinkByName(string) (Link, error)
	LinkAdd(Link) error
	LinkDel(Link) error
	LinkSetName(Link, string) error
	LinkSetMTU(Link, int) error
	LinkSetMaster(Link, *Bridge) error
	LinkSetUp(Link) error
	LinkSetDown(Link) error
	LinkSetNsFd(Link, int) error
	LinkSetHardwareAddr(Link, net.HardwareAddr) error
	NeighSet(*Neigh) error
	Delete()
}

func NewHandleAt(ns netns.NsHandle, nlFamilies ...int) (*netlink.Handle, error) {
	return netlink.NewHandleAt(ns, nlFamilies...)
}
