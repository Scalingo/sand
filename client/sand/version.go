package sand

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/pkg/errors"
)

func (c *client) Version(ctx context.Context) (string, error) {
	url := fmt.Sprintf("%s/version", c.url)
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return "", errors.Wrapf(err, "fail to create http request")
	}
	req = req.WithContext(ctx)
	res, err := c.httpClient.Do(req)
	if err != nil {
		return "", errors.Wrapf(err, "fail to execute GET /version")
	}
	defer res.Body.Close()

	var r map[string]string
	err = json.NewDecoder(res.Body).Decode(&r)
	if err != nil {
		return "", errors.Wrapf(err, "fail to unserialize JSON")
	}

	return r["version"], nil

}
