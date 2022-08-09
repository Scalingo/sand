package docker

import (
	"context"

	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"

	"github.com/Scalingo/go-plugins-helpers/network"
	"github.com/Scalingo/go-utils/logger"
	"github.com/Scalingo/sand/api/params"
	"github.com/Scalingo/sand/endpoint"
	sandnetwork "github.com/Scalingo/sand/network"
)

type dockerNetworkPlugin struct {
	networkRepository      sandnetwork.Repository
	endpointRepository     endpoint.Repository
	dockerPluginRepository Repository
}

func (p *dockerNetworkPlugin) GetCapabilities(ctx context.Context) (*network.CapabilitiesResponse, error) {
	return &network.CapabilitiesResponse{
		ConnectivityScope: network.GlobalScope,
		Scope:             network.LocalScope,
	}, nil
}

func (p *dockerNetworkPlugin) CreateNetwork(ctx context.Context, req *network.CreateNetworkRequest) error {
	log := logger.Get(ctx)
	log.Info("Create network by docker integration")

	opts, ok := req.Options["com.docker.network.generic"].(map[string]interface{})
	if !ok {
		return errors.Errorf("invalid generic options: %+v, not a map[string]interface{}", req.Options["com.docker.network.generic"])
	}

	id, ok := opts["sand-id"].(string)
	if !ok {
		return errors.New("sand-id should be a string")
	}
	log = log.WithField("network_id", id)
	ctx = logger.ToCtx(ctx, log)

	network, ok, err := p.networkRepository.Exists(ctx, id)
	if err != nil {
		return errors.Wrapf(err, "fail to get network %v", id)
	}
	if !ok {
		return errors.Errorf("SAND network %v does not exist", id)
	}

	err = p.dockerPluginRepository.SaveNetwork(ctx, DockerPluginNetwork{
		SandNetworkID:   network.ID,
		DockerNetworkID: req.NetworkID,
	})
	if err != nil {
		return errors.Wrapf(err, "fail to create docker network binding")
	}
	log.Info("Network created by docker integration")
	return nil
}

func (p *dockerNetworkPlugin) AllocateNetwork(ctx context.Context, req *network.AllocateNetworkRequest) (*network.AllocateNetworkResponse, error) {
	return nil, errors.New("unsupported")
}

func (p *dockerNetworkPlugin) DeleteNetwork(ctx context.Context, req *network.DeleteNetworkRequest) error {
	log := logger.Get(ctx).WithField("docker_network_id", req.NetworkID)
	ctx = logger.ToCtx(ctx, log)
	log.Info("Delete network by docker integration")

	dpn, err := p.dockerPluginRepository.GetNetworkByDockerID(ctx, req.NetworkID)
	if err != nil {
		return errors.Wrapf(err, "fail to get docker id binding")
	}

	log = log.WithField("network_id", dpn.SandNetworkID)
	ctx = logger.ToCtx(ctx, log)

	network, ok, err := p.networkRepository.Exists(ctx, dpn.SandNetworkID)
	if err != nil {
		return errors.Wrapf(err, "fail to get network %v", dpn.SandNetworkID)
	}
	if !ok {
		return errors.New("sand network not found")
	}

	err = p.networkRepository.Deactivate(ctx, network)
	if err != nil {
		return errors.Wrapf(err, "fail to deactivate sand network %v", dpn.SandNetworkID)
	}

	err = p.dockerPluginRepository.DeleteNetwork(ctx, dpn)
	if err != nil {
		return errors.Wrapf(err, "fail to delete network docker binding %v", dpn)
	}

	log.Info("Network deactivated and deleted by docker integration")
	return nil
}

func (p *dockerNetworkPlugin) FreeNetwork(ctx context.Context, req *network.FreeNetworkRequest) error {
	return nil
}

func (p *dockerNetworkPlugin) CreateEndpoint(ctx context.Context, req *network.CreateEndpointRequest) (*network.CreateEndpointResponse, error) {
	log := logger.Get(ctx).WithField("docker_network_id", req.NetworkID)
	ctx = logger.ToCtx(ctx, log)
	log.Info("Create endpoint by docker integration")
	dpn, err := p.dockerPluginRepository.GetNetworkByDockerID(ctx, req.NetworkID)
	if err != nil {
		return nil, errors.Wrapf(err, "fail to get docker id binding")
	}

	log = log.WithField("network_id", dpn.SandNetworkID)
	ctx = logger.ToCtx(ctx, log)
	n, ok, err := p.networkRepository.Exists(ctx, dpn.SandNetworkID)
	if err != nil {
		return nil, errors.Wrapf(err, "fail to get network %v", dpn.SandNetworkID)
	}
	if !ok {
		return nil, errors.New("sand network not found")
	}

	params := params.EndpointCreate{
		NetworkID: n.ID,
	}

	if req.Interface.Address != "" {
		params.IPv4Address = req.Interface.Address
		params.MacAddress = req.Interface.MacAddress
	}

	e, err := p.endpointRepository.Create(ctx, n, params)
	if err != nil {
		return nil, errors.Wrapf(err, "fail to create endpoint")
	}
	log = log.WithField("endpoint_id", e.ID)
	ctx = logger.ToCtx(ctx, log)

	err = p.dockerPluginRepository.SaveEndpoint(ctx, DockerPluginEndpoint{
		DockerPluginNetwork: dpn,
		DockerEndpointID:    req.EndpointID,
		SandEndpointID:      e.ID,
	})
	if err != nil {
		return nil, errors.Wrapf(err, "fail to save docker plugin endpoint")
	}

	res := &network.CreateEndpointResponse{Interface: &network.EndpointInterface{}}
	if params.IPv4Address == "" {
		res.Interface.Address = e.TargetVethIP
		res.Interface.MacAddress = e.TargetVethMAC
	}

	log.Info("Endpoint created by docker integration")
	return res, nil
}

func (p *dockerNetworkPlugin) DeleteEndpoint(ctx context.Context, req *network.DeleteEndpointRequest) error {
	log := logger.Get(ctx).WithField("docker_endpoint_id", req.EndpointID)
	log.Info("Delete endpoint by docker integration")

	dpe, err := p.dockerPluginRepository.GetEndpointByDockerID(ctx, req.EndpointID)
	if err != nil {
		return errors.Wrapf(err, "fail to get docker id binding")
	}

	log = log.WithFields(logrus.Fields{
		"endpoint_id": dpe.SandEndpointID,
		"network_id":  dpe.SandNetworkID,
	})
	ctx = logger.ToCtx(ctx, log)

	n, ok, err := p.networkRepository.Exists(ctx, dpe.SandNetworkID)
	if err != nil {
		return errors.Wrapf(err, "fail to get network %v", dpe.SandNetworkID)
	}
	if !ok {
		return errors.New("sand network not found")
	}

	e, ok, err := p.endpointRepository.Exists(ctx, dpe.SandEndpointID)
	if err != nil {
		return errors.Wrapf(err, "fail to get endpoint %v", dpe.SandNetworkID)
	}
	if !ok {
		return errors.New("sand endpoint not found")
	}

	err = p.endpointRepository.Delete(ctx, n, e, endpoint.DeleteOpts{})
	if err != nil {
		return errors.Wrapf(err, "fail to delete endpoint")
	}

	err = p.dockerPluginRepository.DeleteEndpoint(ctx, dpe)
	if err != nil {
		return errors.Wrapf(err, "fail to delete endpoint docker binding of %v", dpe)
	}

	log.Info("Endpoint deleted by docker registry")
	return nil
}

func (p *dockerNetworkPlugin) EndpointInfo(ctx context.Context, req *network.InfoRequest) (*network.InfoResponse, error) {
	return nil, nil
}

func (p *dockerNetworkPlugin) Join(ctx context.Context, req *network.JoinRequest) (*network.JoinResponse, error) {
	dpe, err := p.dockerPluginRepository.GetEndpointByDockerID(ctx, req.EndpointID)
	if err != nil {
		return nil, errors.Wrapf(err, "fail to get docker id binding")
	}

	n, ok, err := p.networkRepository.Exists(ctx, dpe.SandNetworkID)
	if err != nil {
		return nil, errors.Wrapf(err, "fail to get network %v", dpe.SandNetworkID)
	}
	if !ok {
		return nil, errors.New("sand network not found")
	}

	e, ok, err := p.endpointRepository.Exists(ctx, dpe.SandEndpointID)
	if err != nil {
		return nil, errors.Wrapf(err, "fail to get endpoint %v", dpe.SandNetworkID)
	}
	if !ok {
		return nil, errors.New("sand endpoint not found")
	}

	err = p.networkRepository.Ensure(ctx, n)
	if err != nil {
		return nil, errors.Wrapf(err, "fail to ensure network")
	}

	e, err = p.endpointRepository.Activate(ctx, n, e, params.EndpointActivate{
		NSHandlePath: req.SandboxKey,
		MoveVeth:     false,
		SetAddr:      false,
	})
	if err != nil {
		return nil, errors.Wrapf(err, "fail to activate endpoint")
	}

	return &network.JoinResponse{InterfaceName: network.InterfaceName{
		SrcName:   e.TargetVethName,
		DstPrefix: "sand",
	}}, nil
}

func (p *dockerNetworkPlugin) Leave(ctx context.Context, req *network.LeaveRequest) error {
	dpe, err := p.dockerPluginRepository.GetEndpointByDockerID(ctx, req.EndpointID)
	if err != nil {
		return errors.Wrapf(err, "fail to get docker id binding")
	}

	n, ok, err := p.networkRepository.Exists(ctx, dpe.SandNetworkID)
	if err != nil {
		return errors.Wrapf(err, "fail to get network %v", dpe.SandNetworkID)
	}
	if !ok {
		return errors.New("sand network not found")
	}

	e, ok, err := p.endpointRepository.Exists(ctx, dpe.SandEndpointID)
	if err != nil {
		return errors.Wrapf(err, "fail to get endpoint %v", dpe.SandNetworkID)
	}
	if !ok {
		return errors.New("sand endpoint not found")
	}

	_, err = p.endpointRepository.Deactivate(ctx, n, e)
	if err != nil {
		return errors.Wrapf(err, "fail to delete endpoint")
	}
	return nil
}

func (p *dockerNetworkPlugin) DiscoverNew(ctx context.Context, req *network.DiscoveryNotification) error {
	return nil
}

func (p *dockerNetworkPlugin) DiscoverDelete(ctx context.Context, req *network.DiscoveryNotification) error {
	return nil
}

func (p *dockerNetworkPlugin) ProgramExternalConnectivity(ctx context.Context, req *network.ProgramExternalConnectivityRequest) error {
	return nil
}

func (p *dockerNetworkPlugin) RevokeExternalConnectivity(ctx context.Context, req *network.RevokeExternalConnectivityRequest) error {
	return nil
}
