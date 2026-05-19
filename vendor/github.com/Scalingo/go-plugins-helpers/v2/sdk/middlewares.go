package sdk

import (
	"encoding/json"
	"net/http"

	"github.com/urfave/negroni/v3"

	"github.com/Scalingo/go-handlers"
)

// contentTypeMiddleware sets the HTTP header `Content-Type` to the content type accepted and sent by Docker plugins
var contentTypeMiddleware = handlers.MiddlewareFunc(func(handler handlers.HandlerFunc) handlers.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request, vars map[string]string) error {
		rw := negroni.NewResponseWriter(w)
		rw.Header().Set("Content-Type", DefaultContentTypeV1_1)
		return handler(rw, r, vars)
	}
})

// ErrorResponse is a formatted error message that Docker can understand
type ErrorResponse struct {
	Err string
}

// NewErrorResponse creates an ErrorResponse with the provided error
func NewErrorResponse(err error) ErrorResponse {
	return ErrorResponse{Err: err.Error()}
}

// errorMiddleware encodes in JSON the error returned by the handler to the HTTP request body
var errorMiddleware = handlers.MiddlewareFunc(func(handler handlers.HandlerFunc) handlers.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request, vars map[string]string) error {
		rw := negroni.NewResponseWriter(w)
		err := handler(rw, r, vars)
		if err != nil {
			if rw.Status() == 0 {
				rw.WriteHeader(http.StatusInternalServerError)
			}
			json.NewEncoder(rw).Encode(NewErrorResponse(err))
			return err
		}

		return nil
	}
})
