package handlers

import (
	"context"
	"net/http"

	"github.com/satori/go.uuid"
)

func RequestIDMiddleware(next HandlerFunc) HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request, vars map[string]string) error {
		id := r.Header.Get("X-Request-ID")
		if len(id) == 0 {
			id = uuid.NewV4().String()
			r.Header.Set("X-Request-ID", id)
		}
		r = r.WithContext(context.WithValue(r.Context(), "request_id", id))
		return next(w, r, vars)
	}
}
