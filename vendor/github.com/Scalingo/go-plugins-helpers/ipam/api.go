package ipam

import (
	"context"
	"net/http"

	"github.com/Scalingo/go-plugins-helpers/sdk"
	"github.com/Scalingo/go-utils/logger"
	"github.com/sirupsen/logrus"
)

const (
	manifest = `{"Implements": ["IpamDriver"]}`

	capabilitiesPath   = "/IpamDriver.GetCapabilities"
	addressSpacesPath  = "/IpamDriver.GetDefaultAddressSpaces"
	requestPoolPath    = "/IpamDriver.RequestPool"
	releasePoolPath    = "/IpamDriver.ReleasePool"
	requestAddressPath = "/IpamDriver.RequestAddress"
	releaseAddressPath = "/IpamDriver.ReleaseAddress"
)

// Ipam represent the interface a driver must fulfill.
type Ipam interface {
	GetCapabilities(context.Context) (*CapabilitiesResponse, error)
	GetDefaultAddressSpaces(context.Context) (*AddressSpacesResponse, error)
	RequestPool(context.Context, *RequestPoolRequest) (*RequestPoolResponse, error)
	ReleasePool(context.Context, *ReleasePoolRequest) error
	RequestAddress(context.Context, *RequestAddressRequest) (*RequestAddressResponse, error)
	ReleaseAddress(context.Context, *ReleaseAddressRequest) error
}

// CapabilitiesResponse returns whether or not this IPAM required pre-made MAC
type CapabilitiesResponse struct {
	RequiresMACAddress bool
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
	Options map[string]string
}

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

// ErrorResponse is a formatted error message that libnetwork can understand
type ErrorResponse struct {
	Err string
}

// NewErrorResponse creates an ErrorResponse with the provided message
func NewErrorResponse(msg string) *ErrorResponse {
	return &ErrorResponse{Err: msg}
}

// Handler forwards requests and responses between the docker daemon and the plugin.
type Handler struct {
	ipam Ipam
	sdk.Handler
}

// NewHandler initializes the request handler with a driver implementation.
func NewHandler(logger logrus.FieldLogger, driver Ipam) *Handler {
	h := &Handler{driver, sdk.NewHandler(logger, manifest)}
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
			sdk.EncodeResponse(w, NewErrorResponse(err.Error()), true)
			return err
		}
		sdk.EncodeResponse(w, res, false)
		return nil
	})
	h.HandleFunc(addressSpacesPath, func(w http.ResponseWriter, r *http.Request, p map[string]string) error {
		res, err := h.ipam.GetDefaultAddressSpaces(r.Context())
		if err != nil {
			sdk.EncodeResponse(w, NewErrorResponse(err.Error()), true)
			return err
		}
		sdk.EncodeResponse(w, res, false)
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
			sdk.EncodeResponse(w, NewErrorResponse(err.Error()), true)
			return err
		}
		sdk.EncodeResponse(w, res, false)
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
			sdk.EncodeResponse(w, NewErrorResponse(err.Error()), true)
			return err
		}
		sdk.EncodeResponse(w, struct{}{}, false)
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
			sdk.EncodeResponse(w, NewErrorResponse(err.Error()), true)
			return err
		}
		sdk.EncodeResponse(w, res, false)
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
			sdk.EncodeResponse(w, NewErrorResponse(err.Error()), true)
			return err
		}
		sdk.EncodeResponse(w, struct{}{}, false)
		return nil
	})
}
