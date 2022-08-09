package network

import (
	"context"
	"fmt"
	"path/filepath"
	"regexp"
	"time"

	"github.com/gofrs/uuid"
	"github.com/pkg/errors"

	"github.com/Scalingo/go-utils/logger"
	"github.com/Scalingo/sand/api/params"
	"github.com/Scalingo/sand/api/types"
	"github.com/Scalingo/sand/idmanager"
	"github.com/Scalingo/sand/network/overlay"
)

var (
	uuidRegexp = regexp.MustCompile("(?i)^[0-9A-F]{8}-[0-9A-F]{4}-4[0-9A-F]{3}-[89AB][0-9A-F]{3}-[0-9A-F]{12}$")
)

func (r *repository) Create(ctx context.Context, params params.NetworkCreate) (types.Network, error) {
	var err error
	log := logger.Get(ctx).WithField("network_name", params.Name)
	log.Info("Create network")

	if params.Type == "" {
		params.Type = types.OverlayNetworkType
	}

	uuid := uuid.Must(uuid.NewV4()).String()
	if params.ID != "" {
		if !uuidRegexp.MatchString(params.ID) {
			return types.Network{}, errors.Errorf("invalid UUID %v", params.ID)
		}
		uuid = params.ID
	}

	if params.Name == "" {
		params.Name = fmt.Sprintf("net-sc-%s", uuid)
	}

	network := types.Network{
		CreatedAt: time.Now(),
		ID:        uuid,
		IPRange:   params.IPRange,
		Gateway:   params.Gateway,
		Name:      params.Name,
		Type:      params.Type,
		NSHandlePath: filepath.Join(
			r.config.NetnsPath, fmt.Sprintf("%s%s", r.config.NetnsPrefix, uuid),
		),
	}
	log = log.WithField("network_id", network.ID)
	ctx = logger.ToCtx(ctx, log)

	vniGen := overlay.NewVNIGenerator(ctx, r.config, r.store)
	var idlock idmanager.Unlocker
	switch network.Type {
	case types.OverlayNetworkType:
		idlock, err = vniGen.Lock(ctx)
		if err != nil {
			return network, errors.Wrapf(err, "fail to lock VNI generator for %s", network)
		}
		vni, err := vniGen.Generate(ctx)
		if err != nil {
			return network, errors.Wrapf(err, "fail to generate VNI for %s", network)
		}

		log.Debugf("vni is %v", vni)
		network.VxLANVNI = vni
	default:
		return network, errors.New("invalid network type for init")
	}

	err = r.store.Set(ctx, network.StorageKey(), &network)
	if err != nil {
		return network, errors.Wrapf(err, "fail to get network %s from store", network)
	}

	if idlock != nil {
		err := idlock.Unlock(ctx)
		if err != nil {
			log.WithError(err).Errorf("fail to unlock VNI generator for %s", network)
		}
	}

	log.Info("Network created")
	return network, nil
}
