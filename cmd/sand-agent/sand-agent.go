package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"path/filepath"

	"github.com/Scalingo/go-handlers"
	"github.com/Scalingo/go-internal-tools/logger"
	"github.com/Scalingo/sand/api/types"
	"github.com/Scalingo/sand/config"
	"github.com/Scalingo/sand/endpoint"
	"github.com/Scalingo/sand/network"
	"github.com/Scalingo/sand/network/overlay"
	"github.com/Scalingo/sand/store"
	"github.com/Scalingo/sand/web"
	"github.com/docker/docker/pkg/reexec"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

func main() {
	log := logger.Default()
	log.SetLevel(logrus.DebugLevel)
	ctx := logger.ToCtx(context.Background(), log)

	// If reexec to create network namespace
	if filepath.Base(os.Args[0]) != "sand" {
		log.WithField("args", os.Args).Info("reexec")
	}
	ok := reexec.Init()
	if ok {
		log.WithField("args", os.Args).Info("reexec done")
		return
	}

	c, err := config.Build()
	if err != nil {
		log.WithError(err).Error("fail to generate initial config")
		os.Exit(-1)
	}

	err = c.CreateDirectories()
	if err != nil {
		log.WithError(err).Error("fail to create runtime directories")
		os.Exit(-1)
	}

	store := store.New(c)
	peerListener := overlay.NewNetworkEndpointListener(c, store)

	err = ensureNetworks(ctx, c, peerListener)
	if err != nil {
		log.WithError(err).Error("fail to ensure existing networks")
		os.Exit(-1)
	}

	r := handlers.NewRouter(log)
	r.Use(handlers.ErrorMiddleware)

	nctrl := web.NewNetworksController(c, peerListener)
	ectrl := web.NewEndpointsController(c, peerListener)

	r.HandleFunc("/networks", nctrl.List).Methods("GET")
	r.HandleFunc("/networks", nctrl.Create).Methods("POST")
	r.HandleFunc("/networks/{id}", nctrl.Destroy).Methods("DELETE")
	r.HandleFunc("/endpoints", ectrl.Create).Methods("POST")

	log.WithField("port", c.HttpPort).Info("Listening")
	http.ListenAndServe(fmt.Sprintf(":%d", c.HttpPort), r)
}

func ensureNetworks(ctx context.Context, c *config.Config, listener overlay.NetworkEndpointListener) error {
	log := logger.Get(ctx)
	ctx = logger.ToCtx(ctx, log)

	log.Info("ensure networks on node")

	s := store.New(c)
	var networks []types.Network
	err := s.Get(ctx, fmt.Sprintf("/nodes/%s/networks/", c.PublicHostname), true, &networks)
	if err == store.ErrNotFound {
		return nil
	}
	if err != nil {
		return errors.Wrapf(err, "fail to get existing networks on %v", c.PublicHostname)
	}

	repo := network.NewRepository(c, s, listener)
	erepo := endpoint.NewRepository(c, s)

	for _, network := range networks {
		log = log.WithField("network_id", network.ID)

		err = s.Get(ctx, network.StorageKey(), false, &network)
		if err != nil {
			log.WithError(err).Error("fail to get network details")
			continue
		}

		log = log.WithField("network_name", network.Name)
		ctx = logger.ToCtx(ctx, log)

		log.Info("ensuring network is setup")
		err = repo.Ensure(ctx, network)
		if err != nil {
			log.WithError(err).Error("fail to ensure network")
			continue
		}

		var endpoints []types.Endpoint
		err = s.Get(ctx, network.EndpointsStorageKey(c.PublicHostname), true, &endpoints)
		if err == store.ErrNotFound {
			continue
		}
		if err != nil {
			log.WithError(err).Error("fail to list network endpoints")
			continue
		}

		log.Info("insuring network endpoints are setup")
		for _, endpoint := range endpoints {
			log = log.WithFields(logrus.Fields{
				"endpoint_id": endpoint.ID, "endpoint_netns_path": endpoint.TargetNetnsPath,
			})
			log.Info("restoring endpoint")
			ctx = logger.ToCtx(ctx, log)
			endpoint, err = erepo.Ensure(ctx, network, endpoint)
			if err != nil {
				log.WithError(err).Error("fail to ensure endpoint")
				continue
			}
		}
	}
	return nil
}

// nlnh := &netlink.Neigh{
// 	IP:           dstIP,
// 	HardwareAddr: dstMac,
// 	State:        netlink.NUD_PERMANENT,
// 	Family:       nh.family,
// }
// if err := nlh.NeighSet(nlnh); err != nil {
// 	return fmt.Errorf("could not add neighbor entry:%+v error:%v", nlnh, err)
// }

// func (n *network) watchMiss(nlSock *nl.NetlinkSocket) {
// 	t := time.Now()
// 	for {
// 		msgs, err := nlSock.Receive()
// 		if err != nil {
// 			n.Lock()
// 			nlFd := nlSock.GetFd()
// 			n.Unlock()
// 			if nlFd == -1 {
// 				// The netlink socket got closed, simply exit to not leak this goroutine
// 				return
// 			}
// 			// When the receive timeout expires the receive will return EAGAIN
// 			if err == syscall.EAGAIN {
// 				// we continue here to avoid spam for timeouts
// 				continue
// 			}
// 			logrus.Errorf("Failed to receive from netlink: %v ", err)
// 			continue
// 		}

// 		for _, msg := range msgs {
// 			if msg.Header.Type != syscall.RTM_GETNEIGH && msg.Header.Type != syscall.RTM_NEWNEIGH {
// 				continue
// 			}

// 			neigh, err := netlink.NeighDeserialize(msg.Data)
// 			if err != nil {
// 				logrus.Errorf("Failed to deserialize netlink ndmsg: %v", err)
// 				continue
// 			}

// 			var (
// 				ip             net.IP
// 				mac            net.HardwareAddr
// 				l2Miss, l3Miss bool
// 			)
// 			if neigh.IP.To4() != nil {
// 				ip = neigh.IP
// 				l3Miss = true
// 			} else if neigh.HardwareAddr != nil {
// 				mac = []byte(neigh.HardwareAddr)
// 				ip = net.IP(mac[2:])
// 				l2Miss = true
// 			} else {
// 				continue
// 			}

// 			// Not any of the network's subnets. Ignore.
// 			if !n.contains(ip) {
// 				continue
// 			}

// 			if neigh.State&(netlink.NUD_STALE|netlink.NUD_INCOMPLETE) == 0 {
// 				continue
// 			}

// 			if n.driver.isSerfAlive() {
// 				logrus.Debugf("miss notification: dest IP %v, dest MAC %v", ip, mac)
// 				mac, IPmask, vtep, err := n.driver.resolvePeer(n.id, ip)
// 				if err != nil {
// 					logrus.Errorf("could not resolve peer %q: %v", ip, err)
// 					continue
// 				}
// 				n.driver.peerAdd(n.id, "dummy", ip, IPmask, mac, vtep, l2Miss, l3Miss, false)
// 			} else if l3Miss && time.Since(t) > time.Second {
// 				// All the local peers will trigger a miss notification but this one is expected and the local container will reply
// 				// autonomously to the ARP request
// 				// In case the gc_thresh3 values is low kernel might reject new entries during peerAdd. This will trigger the following
// 				// extra logs that will inform of the possible issue.
// 				// Entries created would not be deleted see documentation http://man7.org/linux/man-pages/man7/arp.7.html:
// 				// Entries which are marked as permanent are never deleted by the garbage-collector.
// 				// The time limit here is to guarantee that the dbSearch is not
// 				// done too frequently causing a stall of the peerDB operations.
// 				pKey, pEntry, err := n.driver.peerDbSearch(n.id, ip)
// 				if err == nil && !pEntry.isLocal {
// 					t = time.Now()
// 					logrus.Warnf("miss notification for peer:%+v l3Miss:%t l2Miss:%t, if the problem persist check the gc_thresh on the host pKey:%+v pEntry:%+v err:%v",
// 						neigh, l3Miss, l2Miss, *pKey, *pEntry, err)
// 				}
// 			}
// 		}
// 	}
// }
