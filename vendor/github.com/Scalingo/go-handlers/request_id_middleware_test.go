package handlers

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/satori/go.uuid"
	"github.com/stretchr/testify/assert"
)

func TestRequestIDMiddleware(t *testing.T) {
	examples := []struct {
		Name              string
		AddHeader         bool
		RequestID         string
		ExpectIdenticalID bool
	}{
		{
			Name:              "request with a X-Request-ID header",
			AddHeader:         true,
			RequestID:         uuid.NewV4().String(),
			ExpectIdenticalID: true,
		}, {
			Name:              "request without a X-Request-ID header",
			AddHeader:         false,
			RequestID:         "",
			ExpectIdenticalID: false,
		}, {
			Name:              "request with an empty X-Request-ID header",
			AddHeader:         true,
			RequestID:         "",
			ExpectIdenticalID: false,
		},
	}

	for _, example := range examples {
		t.Run(example.Name, func(t *testing.T) {
			req, err := http.NewRequest("GET", "/", nil)
			assert.NoError(t, err)

			if example.AddHeader {
				req.Header.Set("X-Request-ID", example.RequestID)
			}

			handler := RequestIDMiddleware(HandlerFunc(func(w http.ResponseWriter, r *http.Request, vars map[string]string) error {
				id := r.Header.Get("X-Request-ID")
				if example.ExpectIdenticalID {
					assert.Equal(t, example.RequestID, id)
				}
				assert.NotEmpty(t, id)
				ctxValue := r.Context().Value("request_id").(string)
				assert.Equal(t, id, ctxValue)

				return nil
			}))

			res := httptest.NewRecorder()
			err = handler(res, req, map[string]string{})
			assert.NoError(t, err)
		})
	}
}
