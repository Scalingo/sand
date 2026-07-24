package sand

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/Scalingo/go-utils/errors/v3"
	"github.com/Scalingo/sand/api/httpresp"
	"github.com/Scalingo/sand/api/params"
	"github.com/Scalingo/sand/api/types"
)

func (c *client) EndpointCreate(ctx context.Context, params params.EndpointCreate) (types.Endpoint, error) {
	var (
		endpoint types.Endpoint
		buffer   = new(bytes.Buffer)
	)
	err := json.NewEncoder(buffer).Encode(&params)
	if err != nil {
		return endpoint, errors.Wrapf(ctx, err, "serialize JSON")
	}
	req, err := http.NewRequest("POST", fmt.Sprintf("%s/endpoints", c.url), buffer)
	if err != nil {
		return endpoint, errors.Wrapf(ctx, err, "create http request")
	}
	res, err := c.httpClient.Do(ctx, req)
	if err != nil {
		return endpoint, errors.Wrapf(ctx, err, "execute POST /endpoints")
	}
	defer res.Body.Close()

	if res.StatusCode != 201 {
		var reserr httpresp.Error
		err = json.NewDecoder(res.Body).Decode(&reserr)
		if err != nil {
			return endpoint, errors.Wrapf(ctx, err, "unserialize JSON")
		}
		return endpoint, reserr
	}

	var r httpresp.EndpointCreate
	err = json.NewDecoder(res.Body).Decode(&r)
	if err != nil {
		return endpoint, errors.Wrapf(ctx, err, "unserialize JSON")
	}

	return r.Endpoint, nil
}

func (c *client) EndpointsList(ctx context.Context, params params.EndpointsList) ([]types.Endpoint, error) {
	url := fmt.Sprintf("%s/endpoints?network_id=%s&hostname=%s", c.url, params.NetworkID, params.Hostname)
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, errors.Wrapf(ctx, err, "create http request")
	}
	res, err := c.httpClient.Do(ctx, req)
	if err != nil {
		return nil, errors.Wrapf(ctx, err, "execute GET /endpoints")
	}
	defer res.Body.Close()

	var r httpresp.EndpointsList
	err = json.NewDecoder(res.Body).Decode(&r)
	if err != nil {
		return nil, errors.Wrapf(ctx, err, "unserialize JSON")
	}

	return r.Endpoints, nil
}

func (c *client) EndpointDelete(ctx context.Context, id string) error {
	req, err := http.NewRequest("DELETE", fmt.Sprintf("%s/endpoints/%s", c.url, id), nil)
	if err != nil {
		return errors.Wrapf(ctx, err, "create http request")
	}
	res, err := c.httpClient.Do(ctx, req)
	if err != nil {
		return errors.Wrapf(ctx, err, "execute DELETE /endpoints/%s", id)
	}
	defer res.Body.Close()

	if res.StatusCode != 204 {
		var reserr httpresp.Error
		err = json.NewDecoder(res.Body).Decode(&reserr)
		if err != nil {
			return errors.Wrapf(ctx, err, "unserialize JSON")
		}
		return reserr
	}

	return nil
}
