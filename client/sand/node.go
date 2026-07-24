package sand

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/Scalingo/go-utils/errors/v3"

	"github.com/Scalingo/sand/api/httpresp"
)

func (c *client) NodeEnsureNetworkEndpoints(ctx context.Context) error {
	req, err := http.NewRequest("POST", c.url+"/node/ensure-network-endpoints", nil)
	if err != nil {
		return errors.Wrapf(ctx, err, "create HTTP request")
	}
	res, err := c.httpClient.Do(ctx, req)
	if err != nil {
		return errors.Wrap(ctx, err, "execute POST /node/ensure-network-endpoints")
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
