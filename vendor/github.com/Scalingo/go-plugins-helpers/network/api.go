package network

import (
	"context"
	"errors"
	"net/http"

	"github.com/sirupsen/logrus"

	"github.com/Scalingo/go-plugins-helpers/sdk"
	"github.com/Scalingo/go-utils/logger"
)

const (
	manifest = `{"Implements": ["NetworkDriver"]}`
	// LocalScope is the correct scope response for a local scope driver
	LocalScope = `local`
	// GlobalScope is the correct scope response for a global scope driver
	GlobalScope = `global`

	capabilitiesPath    = "/NetworkDriver.GetCapabilities"
	allocateNetworkPath = "/NetworkDriver.AllocateNetwork"
	freeNetworkPath     = "/NetworkDriver.FreeNetwork"
	createNetworkPath   = "/NetworkDriver.CreateNetwork"
	deleteNetworkPath   = "/NetworkDriver.DeleteNetwork"
	createEndpointPath  = "/NetworkDriver.CreateEndpoint"
	endpointInfoPath    = "/NetworkDriver.EndpointOperInfo"
	deleteEndpointPath  = "/NetworkDriver.DeleteEndpoint"
	joinPath            = "/NetworkDriver.Join"
	leavePath           = "/NetworkDriver.Leave"
	discoverNewPath     = "/NetworkDriver.DiscoverNew"
	discoverDeletePath  = "/NetworkDriver.DiscoverDelete"
	programExtConnPath  = "/NetworkDriver.ProgramExternalConnectivity"
	revokeExtConnPath   = "/NetworkDriver.RevokeExternalConnectivity"
)

// Driver represent the interface a driver must fulfill.
type Driver interface {
	GetCapabilities(context.Context) (*CapabilitiesResponse, error)
	CreateNetwork(context.Context, *CreateNetworkRequest) error
	AllocateNetwork(context.Context, *AllocateNetworkRequest) (*AllocateNetworkResponse, error)
	DeleteNetwork(context.Context, *DeleteNetworkRequest) error
	FreeNetwork(context.Context, *FreeNetworkRequest) error
	CreateEndpoint(context.Context, *CreateEndpointRequest) (*CreateEndpointResponse, error)
	DeleteEndpoint(context.Context, *DeleteEndpointRequest) error
	EndpointInfo(context.Context, *InfoRequest) (*InfoResponse, error)
	Join(context.Context, *JoinRequest) (*JoinResponse, error)
	Leave(context.Context, *LeaveRequest) error
	DiscoverNew(context.Context, *DiscoveryNotification) error
	DiscoverDelete(context.Context, *DiscoveryNotification) error
	ProgramExternalConnectivity(context.Context, *ProgramExternalConnectivityRequest) error
	RevokeExternalConnectivity(context.Context, *RevokeExternalConnectivityRequest) error
}

// CapabilitiesResponse returns whether or not this network is global or local
type CapabilitiesResponse struct {
	Scope             string
	ConnectivityScope string
}

// AllocateNetworkRequest requests allocation of new network by manager
type AllocateNetworkRequest struct {
	// A network ID that remote plugins are expected to store for future
	// reference.
	NetworkID string

	// A free form map->object interface for communication of options.
	Options map[string]string

	// IPAMData contains the address pool information for this network
	IPv4Data, IPv6Data []IPAMData
}

// AllocateNetworkResponse is the response to the AllocateNetworkRequest.
type AllocateNetworkResponse struct {
	// A free form plugin specific string->string object to be sent in
	// CreateNetworkRequest call in the libnetwork agents
	Options map[string]string
}

// FreeNetworkRequest is the request to free allocated network in the manager
type FreeNetworkRequest struct {
	// The ID of the network to be freed.
	NetworkID string
}

// CreateNetworkRequest is sent by the daemon when a network needs to be created
type CreateNetworkRequest struct {
	NetworkID string
	Options   map[string]interface{}
	IPv4Data  []*IPAMData
	IPv6Data  []*IPAMData
}

// IPAMData contains IPv4 or IPv6 addressing information
type IPAMData struct {
	AddressSpace string
	Pool         string
	Gateway      string
	AuxAddresses map[string]interface{}
}

// DeleteNetworkRequest is sent by the daemon when a network needs to be removed
type DeleteNetworkRequest struct {
	NetworkID string
}

// CreateEndpointRequest is sent by the daemon when an endpoint should be created
type CreateEndpointRequest struct {
	NetworkID  string
	EndpointID string
	Interface  *EndpointInterface
	Options    map[string]interface{}
}

// CreateEndpointResponse is sent as a response to a CreateEndpointRequest
type CreateEndpointResponse struct {
	Interface *EndpointInterface
}

// EndpointInterface contains endpoint interface information
type EndpointInterface struct {
	Address     string
	AddressIPv6 string
	MacAddress  string
}

// DeleteEndpointRequest is sent by the daemon when an endpoint needs to be removed
type DeleteEndpointRequest struct {
	NetworkID  string
	EndpointID string
}

// InterfaceName consists of the name of the interface in the global netns and
// the desired prefix to be appended to the interface inside the container netns
type InterfaceName struct {
	SrcName   string
	DstPrefix string
}

// InfoRequest is send by the daemon when querying endpoint information
type InfoRequest struct {
	NetworkID  string
	EndpointID string
}

// InfoResponse is endpoint information sent in response to an InfoRequest
type InfoResponse struct {
	Value map[string]string
}

// JoinRequest is sent by the Daemon when an endpoint needs be joined to a network
type JoinRequest struct {
	NetworkID  string
	EndpointID string
	SandboxKey string
	Options    map[string]interface{}
}

// StaticRoute contains static route information
type StaticRoute struct {
	Destination string
	RouteType   int
	NextHop     string
}

// JoinResponse is sent in response to a JoinRequest
type JoinResponse struct {
	InterfaceName         InterfaceName
	Gateway               string
	GatewayIPv6           string
	StaticRoutes          []*StaticRoute
	DisableGatewayService bool
}

// LeaveRequest is send by the daemon when a endpoint is leaving a network
type LeaveRequest struct {
	NetworkID  string
	EndpointID string
}

// ErrorResponse is a formatted error message that libnetwork can understand
type ErrorResponse struct {
	Err string
}

// DiscoveryNotification is sent by the daemon when a new discovery event occurs
type DiscoveryNotification struct {
	DiscoveryType int
	DiscoveryData interface{}
}

// ProgramExternalConnectivityRequest specifies the L4 data
// and the endpoint for which programming has to be done
type ProgramExternalConnectivityRequest struct {
	NetworkID  string
	EndpointID string
	Options    map[string]interface{}
}

// RevokeExternalConnectivityRequest specifies the endpoint
// for which the L4 programming has to be removed
type RevokeExternalConnectivityRequest struct {
	NetworkID  string
	EndpointID string
}

// NewErrorResponse creates an ErrorResponse with the provided message
func NewErrorResponse(msg string) *ErrorResponse {
	return &ErrorResponse{Err: msg}
}

// Handler forwards requests and responses between the docker daemon and the plugin.
type Handler struct {
	driver Driver
	sdk.Handler
}

// NewHandler initializes the request handler with a driver implementation.
func NewHandler(logger logrus.FieldLogger, driver Driver) *Handler {
	h := &Handler{driver, sdk.NewHandler(logger, manifest)}
	h.initMux()
	return h
}

// ConfigureHandler adds routes to the sdk.Handler to handle the network plugin API
func ConfigureHandler(sdkhandler sdk.Handler, driver Driver) {
	h := &Handler{driver, sdkhandler}
	h.initMux()
}

func (h *Handler) initMux() {
	h.HandleFunc(capabilitiesPath, func(w http.ResponseWriter, r *http.Request, p map[string]string) error {
		res, err := h.driver.GetCapabilities(r.Context())
		if err != nil {
			sdk.EncodeResponse(w, NewErrorResponse(err.Error()), true)
			return err
		}
		if res == nil {
			err := errors.New("Network driver must implement GetCapabilities")
			sdk.EncodeResponse(w, NewErrorResponse(err.Error()), true)
			return err
		}
		sdk.EncodeResponse(w, res, false)
		return nil
	})
	h.HandleFunc(createNetworkPath, func(w http.ResponseWriter, r *http.Request, p map[string]string) error {
		req := &CreateNetworkRequest{}
		err := sdk.DecodeRequest(w, r, req)
		if err != nil {
			return err
		}
		ctx := logger.ToCtx(r.Context(), logger.Get(r.Context()).WithFields(logrus.Fields{
			"docker_network_id": req.NetworkID,
		}))
		err = h.driver.CreateNetwork(ctx, req)
		if err != nil {
			sdk.EncodeResponse(w, NewErrorResponse(err.Error()), true)
			return err
		}
		sdk.EncodeResponse(w, struct{}{}, false)
		return nil
	})
	h.HandleFunc(allocateNetworkPath, func(w http.ResponseWriter, r *http.Request, p map[string]string) error {
		req := &AllocateNetworkRequest{}
		err := sdk.DecodeRequest(w, r, req)
		if err != nil {
			return err
		}
		ctx := logger.ToCtx(r.Context(), logger.Get(r.Context()).WithFields(logrus.Fields{
			"docker_network_id": req.NetworkID,
		}))
		res, err := h.driver.AllocateNetwork(ctx, req)
		if err != nil {
			sdk.EncodeResponse(w, NewErrorResponse(err.Error()), true)
			return err
		}
		sdk.EncodeResponse(w, res, false)
		return nil
	})
	h.HandleFunc(deleteNetworkPath, func(w http.ResponseWriter, r *http.Request, p map[string]string) error {
		req := &DeleteNetworkRequest{}
		err := sdk.DecodeRequest(w, r, req)
		if err != nil {
			return err
		}
		ctx := logger.ToCtx(r.Context(), logger.Get(r.Context()).WithFields(logrus.Fields{
			"docker_network_id": req.NetworkID,
		}))
		err = h.driver.DeleteNetwork(ctx, req)
		if err != nil {
			sdk.EncodeResponse(w, NewErrorResponse(err.Error()), true)
			return err
		}
		sdk.EncodeResponse(w, struct{}{}, false)
		return nil
	})
	h.HandleFunc(freeNetworkPath, func(w http.ResponseWriter, r *http.Request, p map[string]string) error {
		req := &FreeNetworkRequest{}
		err := sdk.DecodeRequest(w, r, req)
		if err != nil {
			return err
		}
		ctx := logger.ToCtx(r.Context(), logger.Get(r.Context()).WithFields(logrus.Fields{
			"docker_network_id": req.NetworkID,
		}))
		err = h.driver.FreeNetwork(ctx, req)
		if err != nil {
			sdk.EncodeResponse(w, NewErrorResponse(err.Error()), true)
			return err
		}
		sdk.EncodeResponse(w, struct{}{}, false)
		return nil
	})
	h.HandleFunc(createEndpointPath, func(w http.ResponseWriter, r *http.Request, p map[string]string) error {
		req := &CreateEndpointRequest{}
		err := sdk.DecodeRequest(w, r, req)
		if err != nil {
			return err
		}
		ctx := logger.ToCtx(r.Context(), logger.Get(r.Context()).WithFields(logrus.Fields{
			"docker_network_id":  req.NetworkID,
			"docker_endpoint_id": req.EndpointID,
		}))
		res, err := h.driver.CreateEndpoint(ctx, req)
		if err != nil {
			sdk.EncodeResponse(w, NewErrorResponse(err.Error()), true)
			return err
		}
		sdk.EncodeResponse(w, res, false)
		return nil
	})
	h.HandleFunc(deleteEndpointPath, func(w http.ResponseWriter, r *http.Request, p map[string]string) error {
		req := &DeleteEndpointRequest{}
		err := sdk.DecodeRequest(w, r, req)
		if err != nil {
			return err
		}
		ctx := logger.ToCtx(r.Context(), logger.Get(r.Context()).WithFields(logrus.Fields{
			"docker_network_id":  req.NetworkID,
			"docker_endpoint_id": req.EndpointID,
		}))
		err = h.driver.DeleteEndpoint(ctx, req)
		if err != nil {
			sdk.EncodeResponse(w, NewErrorResponse(err.Error()), true)
			return err
		}
		sdk.EncodeResponse(w, struct{}{}, false)
		return nil
	})
	h.HandleFunc(endpointInfoPath, func(w http.ResponseWriter, r *http.Request, p map[string]string) error {
		req := &InfoRequest{}
		err := sdk.DecodeRequest(w, r, req)
		if err != nil {
			return err
		}
		ctx := logger.ToCtx(r.Context(), logger.Get(r.Context()).WithFields(logrus.Fields{
			"docker_network_id":  req.NetworkID,
			"docker_endpoint_id": req.EndpointID,
		}))
		res, err := h.driver.EndpointInfo(ctx, req)
		if err != nil {
			sdk.EncodeResponse(w, NewErrorResponse(err.Error()), true)
			return err
		}
		sdk.EncodeResponse(w, res, false)
		return nil
	})
	h.HandleFunc(joinPath, func(w http.ResponseWriter, r *http.Request, p map[string]string) error {
		req := &JoinRequest{}
		err := sdk.DecodeRequest(w, r, req)
		if err != nil {
			return err
		}
		ctx := logger.ToCtx(r.Context(), logger.Get(r.Context()).WithFields(logrus.Fields{
			"docker_network_id":  req.NetworkID,
			"docker_endpoint_id": req.EndpointID,
		}))
		res, err := h.driver.Join(ctx, req)
		if err != nil {
			sdk.EncodeResponse(w, NewErrorResponse(err.Error()), true)
			return err
		}
		sdk.EncodeResponse(w, res, false)
		return nil
	})
	h.HandleFunc(leavePath, func(w http.ResponseWriter, r *http.Request, p map[string]string) error {
		req := &LeaveRequest{}
		err := sdk.DecodeRequest(w, r, req)
		if err != nil {
			return err
		}
		ctx := logger.ToCtx(r.Context(), logger.Get(r.Context()).WithFields(logrus.Fields{
			"docker_network_id":  req.NetworkID,
			"docker_endpoint_id": req.EndpointID,
		}))
		err = h.driver.Leave(ctx, req)
		if err != nil {
			sdk.EncodeResponse(w, NewErrorResponse(err.Error()), true)
			return err
		}
		sdk.EncodeResponse(w, struct{}{}, false)
		return nil
	})
	h.HandleFunc(discoverNewPath, func(w http.ResponseWriter, r *http.Request, p map[string]string) error {
		req := &DiscoveryNotification{}
		err := sdk.DecodeRequest(w, r, req)
		if err != nil {
			return err
		}
		err = h.driver.DiscoverNew(r.Context(), req)
		if err != nil {
			sdk.EncodeResponse(w, NewErrorResponse(err.Error()), true)
			return err
		}
		sdk.EncodeResponse(w, struct{}{}, false)
		return nil
	})
	h.HandleFunc(discoverDeletePath, func(w http.ResponseWriter, r *http.Request, p map[string]string) error {
		req := &DiscoveryNotification{}
		err := sdk.DecodeRequest(w, r, req)
		if err != nil {
			return err
		}
		err = h.driver.DiscoverDelete(r.Context(), req)
		if err != nil {
			sdk.EncodeResponse(w, NewErrorResponse(err.Error()), true)
			return err
		}
		sdk.EncodeResponse(w, struct{}{}, false)
		return nil
	})
	h.HandleFunc(programExtConnPath, func(w http.ResponseWriter, r *http.Request, p map[string]string) error {
		req := &ProgramExternalConnectivityRequest{}
		err := sdk.DecodeRequest(w, r, req)
		if err != nil {
			return err
		}
		err = h.driver.ProgramExternalConnectivity(r.Context(), req)
		if err != nil {
			sdk.EncodeResponse(w, NewErrorResponse(err.Error()), true)
			return err
		}
		sdk.EncodeResponse(w, struct{}{}, false)
		return nil
	})
	h.HandleFunc(revokeExtConnPath, func(w http.ResponseWriter, r *http.Request, p map[string]string) error {
		req := &RevokeExternalConnectivityRequest{}
		err := sdk.DecodeRequest(w, r, req)
		if err != nil {
			return err
		}
		err = h.driver.RevokeExternalConnectivity(r.Context(), req)
		if err != nil {
			sdk.EncodeResponse(w, NewErrorResponse(err.Error()), true)
			return err
		}
		sdk.EncodeResponse(w, struct{}{}, false)
		return nil
	})
}
