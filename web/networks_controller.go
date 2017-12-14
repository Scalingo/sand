package web

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"github.com/Scalingo/go-internal-tools/logger"
	"github.com/Scalingo/networking-agent/api/params"
	"github.com/Scalingo/networking-agent/api/types"
	"github.com/Scalingo/networking-agent/config"
	"github.com/Scalingo/networking-agent/netnsbuilder"
	"github.com/docker/libnetwork/ns"
	"github.com/pkg/errors"
	"github.com/vishvananda/netlink"
	"github.com/vishvananda/netns"
)

type NetworksController struct {
	Config *config.Config
}

func NewNetworksController(c *config.Config) NetworksController {
	return NetworksController{Config: c}
}

type NetworkList struct {
	Networks []types.Network `json:"networks"`
}

func (c NetworksController) List(w http.ResponseWriter, r *http.Request, params map[string]string) error {
	w.Header().Set("Content-Type", "application/json")
	log := logger.Get(r.Context())

	f, err := os.Open(c.Config.NetnsPath)
	if err != nil {
		return errors.Wrapf(err, "fail to open netns directory")
	}

	filenames, err := f.Readdirnames(-1)
	if err != nil {
		return errors.Wrapf(err, "fail to list netns handlers in %v", c.Config.NetnsPath)
	}

	log.Debugf("%v namespace handlers are present")
	res := NetworkList{Networks: []types.Network{}}
	for _, n := range filenames {
		ns := filepath.Base(n)
		if strings.HasPrefix(ns, c.Config.NetnsPrefix) {
			name := strings.TrimPrefix(ns, c.Config.NetnsPrefix)
			res.Networks = append(res.Networks, types.Network{NSHandlePath: n, Name: name, Type: types.OverlayNetworkType})
		}
	}

	w.WriteHeader(200)
	err = json.NewEncoder(w).Encode(&res)
	if err != nil {
		log.WithError(err).Error("fail to encode JSON")
	}
	return nil
}

type NetworkCreateRes struct {
	Network types.Network `json:"network"`
}

func (c NetworksController) Create(w http.ResponseWriter, r *http.Request, p map[string]string) error {
	w.Header().Set("Content-Type", "application/json")
	ctx := r.Context()
	log := logger.Get(ctx)

	var cnp params.CreateNetworkParams
	err := json.NewDecoder(r.Body).Decode(&cnp)
	if err != nil {
		return errors.Wrap(err, "invalid JSON")
	}

	network := types.Network{
		CreatedAt: time.Now(),
		Name:      cnp.Name,
		NSHandlePath: filepath.Join(
			c.Config.NetnsPath, fmt.Sprintf("%s%s", c.Config.NetnsPrefix, cnp.Name),
		),
	}

	log = log.WithField("ns-name", cnp.Name)
	ctx = logger.ToCtx(ctx, log)

	m := netnsbuilder.NewManager(c.Config)
	err = m.Create(ctx, network.Name, network)
	if err != nil && err != netnsbuilder.ErrAlreadyExist {
		return errors.Wrapf(err, "fail to create network namspace")
	}

	nsfd, err := netns.GetFromPath(network.NSHandlePath)
	if err != nil {
		return errors.Wrapf(err, "fail to get namespace handler")
	}
	defer nsfd.Close()

	nlh, err := netlink.NewHandleAt(nsfd, syscall.NETLINK_ROUTE)
	if err != nil {
		return errors.Wrapf(err, "fail to get netlink handler of netns")
	}

	links, err := nlh.LinkList()
	if err != nil {
		return errors.Wrapf(err, "fail to list links")
	}

	exist := false
	var bridge *netlink.Bridge
	for _, link := range links {
		if link.Attrs().Name == "br0" {
			bridge = link.(*netlink.Bridge)
			exist = true
			break
		}
	}

	if !exist {
		link := &netlink.Bridge{
			LinkAttrs: netlink.LinkAttrs{
				Name: "br0",
			},
		}

		if err := nlh.LinkAdd(link); err != nil {
			return errors.Wrapf(err, "fail to create bridge in namespace")
		}

		bridgeLink, err := nlh.LinkByName("br0")
		if err != nil {
			return errors.Wrapf(err, "fail to get bridge link")
		}

		bridge = bridgeLink.(*netlink.Bridge)
	}

	exist = false
	for _, link := range links {
		if link.Attrs().Name == "vxlan0" {
			exist = true
			break
		}
	}

	if !exist {
		vxlan := &netlink.Vxlan{
			LinkAttrs: netlink.LinkAttrs{Name: "vxlan0", MTU: 1450},
			VxlanId:   int(1),
			Learning:  true,
			Port:      4789,
			Proxy:     true,
			L3miss:    true,
			L2miss:    true,
		}

		if err := ns.NlHandle().LinkAdd(vxlan); err != nil {
			return errors.Wrap(err, "error creating vxlan interface")
		}

		link, err := ns.NlHandle().LinkByName("vxlan0")
		if err != nil {
			return errors.Wrap(err, "fail to get vxlan0 link")
		}

		err = ns.NlHandle().LinkSetNsFd(link, int(nsfd))
		if err != nil {
			return errors.Wrap(err, "fail to set netns of vxlan")
		}
	}

	link, err := nlh.LinkByName("vxlan0")
	if err != nil {
		return errors.Wrap(err, "fail to get vxlan0 link")
	}

	if link.Attrs().MasterIndex == 0 {
		err := nlh.LinkSetMaster(link, bridge)
		if err != nil {
			return errors.Wrap(err, "fail to set vxlan0 in bridge br0")
		}
	}

	w.WriteHeader(201)
	err = json.NewEncoder(w).Encode(&NetworkCreateRes{
		Network: network,
	})
	if err != nil {
		log.WithError(err).Error("fail to encode JSON")
	}
	return nil
}
