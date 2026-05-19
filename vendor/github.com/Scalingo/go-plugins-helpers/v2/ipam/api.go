package ipam

import (
	"context"
	"net/http"

	"github.com/sirupsen/logrus"

	"github.com/Scalingo/go-plugins-helpers/v2/sdk"
	"github.com/Scalingo/go-utils/logger"
)

const (
	// ImplementationName is the name of the interface all IPAM plugins implement
	ImplementationName sdk.DriverImplementationName = "IpamDriver"

	capabilitiesPath   = "/IpamDriver.GetCapabilities"
	addressSpacesPath  = "/IpamDriver.GetDefaultAddressSpaces"
	requestPoolPath    = "/IpamDriver.RequestPool"
	releasePoolPath    = "/IpamDriver.ReleasePool"
	requestAddressPath = "/IpamDriver.RequestAddress"
	releaseAddressPath = "/IpamDriver.ReleaseAddress"
)

// Ipam represent the interface a driver must fulfill.
type Ipam interface {
	GetCapabilities(ctx context.Context) (*CapabilitiesResponse, error)
	GetDefaultAddressSpaces(ctx context.Context) (*AddressSpacesResponse, error)
	RequestPool(ctx context.Context, req *RequestPoolRequest) (*RequestPoolResponse, error)
	ReleasePool(ctx context.Context, req *ReleasePoolRequest) error
	RequestAddress(ctx context.Context, req *RequestAddressRequest) (*RequestAddressResponse, error)
	ReleaseAddress(ctx context.Context, req *ReleaseAddressRequest) error
}

// CapabilitiesResponse returns whether or not this IPAM required pre-made MAC
type CapabilitiesResponse struct {
	RequiresMACAddress    bool
	RequiresRequestReplay bool
}

// AddressSpacesResponse returns the default local and global address space names for this IPAM
type AddressSpacesResponse struct {
	LocalDefaultAddressSpace  string
	GlobalDefaultAddressSpace string
}

// RequestPoolRequest is sent by the daemon when a pool needs to be created
type RequestPoolRequest struct {
	AddressSpace string
	Pool         string
	SubPool      string
	Options      map[string]string
	V6           bool
}

// RequestPoolResponse returns a registered address pool with the IPAM driver
type RequestPoolResponse struct {
	PoolID string
	Pool   string
	Data   map[string]string
}

// ReleasePoolRequest is sent when releasing a previously registered address pool
type ReleasePoolRequest struct {
	PoolID string
}

// RequestAddressRequest is sent when requesting an address from IPAM
type RequestAddressRequest struct {
	PoolID  string
	Address string
	Options RequestAddressRequestOptions
}

type RequestAddressRequestOptions struct {
	RequestAddressType RequestAddressType `json:"RequestAddressType"`
}

type RequestAddressType string

const (
	RequestAddressTypeGateway RequestAddressType = "com.docker.network.gateway"
)

// RequestAddressResponse is formed with allocated address by IPAM
type RequestAddressResponse struct {
	Address string
	Data    map[string]string
}

// ReleaseAddressRequest is sent in order to release an address from the pool
type ReleaseAddressRequest struct {
	PoolID  string
	Address string
}

// Handler forwards requests and responses between the docker daemon and the plugin.
type Handler struct {
	ipam Ipam
	sdk.Handler
}

// NewHandler initializes the request handler with a driver implementation.
func NewHandler(logger logrus.FieldLogger, driver Ipam) *Handler {
	h := &Handler{driver, sdk.NewHandler(logger, sdk.Manifest{
		Implements: []sdk.DriverImplementationName{ImplementationName},
	})}
	h.initMux()
	return h
}

// ConfigureHandler adds routes to the sdk.Handler to handle the Ipam plugin API
func ConfigureHandler(sdkhandler sdk.Handler, driver Ipam) {
	h := &Handler{driver, sdkhandler}
	h.initMux()
}

func (h *Handler) initMux() {
	h.HandleFunc(capabilitiesPath, func(w http.ResponseWriter, r *http.Request, p map[string]string) error {
		res, err := h.ipam.GetCapabilities(r.Context())
		if err != nil {
			return err
		}
		sdk.EncodeResponse(w, res)
		return nil
	})
	h.HandleFunc(addressSpacesPath, func(w http.ResponseWriter, r *http.Request, p map[string]string) error {
		res, err := h.ipam.GetDefaultAddressSpaces(r.Context())
		if err != nil {
			return err
		}
		sdk.EncodeResponse(w, res)
		return nil
	})
	h.HandleFunc(requestPoolPath, func(w http.ResponseWriter, r *http.Request, p map[string]string) error {
		req := &RequestPoolRequest{}
		err := sdk.DecodeRequest(w, r, req)
		if err != nil {
			return err
		}
		ctx := logger.ToCtx(r.Context(), logger.Get(r.Context()).WithFields(logrus.Fields{
			"docker_address_space": req.AddressSpace,
			"docker_pool":          req.Pool,
			"docker_subpool":       req.SubPool,
		}))
		res, err := h.ipam.RequestPool(ctx, req)
		if err != nil {
			return err
		}
		sdk.EncodeResponse(w, res)
		return nil
	})
	h.HandleFunc(releasePoolPath, func(w http.ResponseWriter, r *http.Request, p map[string]string) error {
		req := &ReleasePoolRequest{}
		err := sdk.DecodeRequest(w, r, req)
		if err != nil {
			return nil
		}
		ctx := logger.ToCtx(r.Context(), logger.Get(r.Context()).WithFields(logrus.Fields{
			"docker_pool_id": req.PoolID,
		}))
		err = h.ipam.ReleasePool(ctx, req)
		if err != nil {
			return err
		}
		sdk.EncodeResponse(w, struct{}{})
		return nil
	})
	h.HandleFunc(requestAddressPath, func(w http.ResponseWriter, r *http.Request, p map[string]string) error {
		req := &RequestAddressRequest{}
		err := sdk.DecodeRequest(w, r, req)
		if err != nil {
			return err
		}
		ctx := logger.ToCtx(r.Context(), logger.Get(r.Context()).WithFields(logrus.Fields{
			"docker_pool_id": req.PoolID,
			"docker_address": req.Address,
		}))
		res, err := h.ipam.RequestAddress(ctx, req)
		if err != nil {
			return err
		}
		sdk.EncodeResponse(w, res)
		return nil
	})
	h.HandleFunc(releaseAddressPath, func(w http.ResponseWriter, r *http.Request, p map[string]string) error {
		req := &ReleaseAddressRequest{}
		err := sdk.DecodeRequest(w, r, req)
		if err != nil {
			return err
		}
		ctx := logger.ToCtx(r.Context(), logger.Get(r.Context()).WithFields(logrus.Fields{
			"docker_pool_id": req.PoolID,
			"docker_address": req.Address,
		}))
		err = h.ipam.ReleaseAddress(ctx, req)
		if err != nil {
			return err
		}
		sdk.EncodeResponse(w, struct{}{})
		return nil
	})
}
