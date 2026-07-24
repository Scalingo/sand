package sand

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/Scalingo/go-utils/errors/v3"
)

func (c *client) Version(ctx context.Context) (string, error) {
	url := fmt.Sprintf("%s/version", c.url)
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return "", errors.Wrapf(ctx, err, "create http request")
	}
	res, err := c.httpClient.Do(ctx, req)
	if err != nil {
		return "", errors.Wrapf(ctx, err, "execute GET /version")
	}
	defer res.Body.Close()

	var r map[string]string
	err = json.NewDecoder(res.Body).Decode(&r)
	if err != nil {
		return "", errors.Wrapf(ctx, err, "unserialize JSON")
	}

	return r["version"], nil

}
