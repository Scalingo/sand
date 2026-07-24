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

func (c *client) NetworkShow(ctx context.Context, id string) (types.Network, error) {
	req, err := http.NewRequest("GET", fmt.Sprintf("%s/networks/%s", c.url, id), nil)
	if err != nil {
		return types.Network{}, errors.Wrapf(ctx, err, "create http request")
	}
	res, err := c.httpClient.Do(ctx, req)
	if err != nil {
		return types.Network{}, errors.Wrapf(ctx, err, "execute GET /networks/%s", id)
	}
	defer res.Body.Close()

	var r httpresp.NetworkShow
	err = json.NewDecoder(res.Body).Decode(&r)
	if err != nil {
		return types.Network{}, errors.Wrapf(ctx, err, "unserialize JSON")
	}

	return r.Network, nil
}

func (c *client) NetworksList(ctx context.Context) ([]types.Network, error) {
	req, err := http.NewRequest("GET", fmt.Sprintf("%s/networks", c.url), nil)
	if err != nil {
		return nil, errors.Wrapf(ctx, err, "create http request")
	}
	res, err := c.httpClient.Do(ctx, req)
	if err != nil {
		return nil, errors.Wrapf(ctx, err, "execute GET /networks")
	}
	defer res.Body.Close()

	var r httpresp.NetworksList
	err = json.NewDecoder(res.Body).Decode(&r)
	if err != nil {
		return nil, errors.Wrapf(ctx, err, "unserialize JSON")
	}

	return r.Networks, nil
}

func (c *client) NetworkCreate(ctx context.Context, params params.NetworkCreate) (types.Network, error) {
	var (
		network types.Network
		buffer  = new(bytes.Buffer)
	)
	err := json.NewEncoder(buffer).Encode(&params)
	if err != nil {
		return network, errors.Wrapf(ctx, err, "serialize JSON")
	}
	req, err := http.NewRequest("POST", fmt.Sprintf("%s/networks", c.url), buffer)
	if err != nil {
		return network, errors.Wrapf(ctx, err, "create http request")
	}
	res, err := c.httpClient.Do(ctx, req)
	if err != nil {
		return network, errors.Wrapf(ctx, err, "execute POST /networks")
	}
	defer res.Body.Close()

	if res.StatusCode != 201 {
		errmap := map[string]interface{}{}
		json.NewDecoder(res.Body).Decode(&errmap)
		return network, errors.Errorf(ctx, "invalid response %v: %v", res.Status, errmap)
	}

	var r httpresp.NetworkCreate
	err = json.NewDecoder(res.Body).Decode(&r)
	if err != nil {
		return network, errors.Wrapf(ctx, err, "unserialize JSON")
	}

	return r.Network, nil
}

func (c *client) NetworkDelete(ctx context.Context, networkID string) error {
	req, err := http.NewRequest("DELETE", fmt.Sprintf("%s/networks/%s", c.url, networkID), nil)
	if err != nil {
		return errors.Wrapf(ctx, err, "create HTTP request")
	}
	res, err := c.httpClient.Do(ctx, req)
	if err != nil {
		return errors.Wrapf(ctx, err, "execute DELETE /networks/%s", networkID)
	}
	defer res.Body.Close()

	if res.StatusCode != http.StatusNoContent {
		var reserr httpresp.Error
		err := json.NewDecoder(res.Body).Decode(&reserr)
		if err != nil {
			return errors.Wrapf(ctx, err, "decode JSON in errors response: %s", res.Status)
		}

		return reserr
	}

	return nil
}
