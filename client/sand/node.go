package sand

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/pkg/errors"

	"github.com/Scalingo/sand/api/httpresp"
)

func (c *client) NodeEnsureNetworkEndpoints(ctx context.Context) error {
	req, err := http.NewRequest("POST", fmt.Sprintf("%s/node/ensure-network-endpoints", c.url), nil)
	if err != nil {
		return errors.Wrapf(err, "failed to create HTTP request")
	}
	req = req.WithContext(ctx)
	res, err := c.httpClient.Do(req)
	if err != nil {
		return errors.Wrap(err, "failed to execute POST /node/ensure-network-endpoints")
	}
	defer res.Body.Close()

	if res.StatusCode != http.StatusNoContent {
		var reserr httpresp.Error
		err := json.NewDecoder(res.Body).Decode(&reserr)
		if err != nil {
			return errors.Wrapf(err, "failed to decode JSON in errors response: %s", res.Status)
		}

		return reserr
	}

	return nil
}
