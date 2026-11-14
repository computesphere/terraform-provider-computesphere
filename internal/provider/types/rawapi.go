package types

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"

	csv2 "github.com/computesphere/computesphere-go"
)

// DoJSON sends a JSON request to a v2 route that the generated SDK doesn't
// cover yet, through the SDK client's base URL, HTTP client and request
// editors (so the bearer token and x-account-id header are set exactly as for
// generated calls, and recorded tests replay through the same transport).
//
// path is relative to the v2 base URL and starts with "/", e.g. "/caches".
// in is marshalled as the request body when non-nil. When out is non-nil and
// the response status is 2xx, the body is unmarshalled into it. The raw body
// and status are always returned so callers can report API problems.
func DoJSON(ctx context.Context, c *csv2.ClientWithResponses, method, path string, query url.Values, in, out any) (int, []byte, error) {
	cl, ok := c.ClientInterface.(*csv2.Client)
	if !ok {
		return 0, nil, fmt.Errorf("unexpected API client type %T", c.ClientInterface)
	}
	base, err := url.Parse(cl.Server)
	if err != nil {
		return 0, nil, err
	}
	u, err := base.Parse("." + "/" + strings.TrimPrefix(path, "/"))
	if err != nil {
		return 0, nil, err
	}
	if len(query) > 0 {
		u.RawQuery = query.Encode()
	}

	var body io.Reader
	if in != nil {
		b, err := json.Marshal(in)
		if err != nil {
			return 0, nil, err
		}
		body = bytes.NewReader(b)
	}
	req, err := http.NewRequestWithContext(ctx, method, u.String(), body)
	if err != nil {
		return 0, nil, err
	}
	if in != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	req.Header.Set("Accept", "application/json")
	for _, edit := range cl.RequestEditors {
		if err := edit(ctx, req); err != nil {
			return 0, nil, err
		}
	}

	resp, err := cl.Client.Do(req)
	if err != nil {
		return 0, nil, err
	}
	defer func() { _ = resp.Body.Close() }()
	raw, err := io.ReadAll(resp.Body)
	if err != nil {
		return resp.StatusCode, nil, err
	}
	if out != nil && resp.StatusCode >= 200 && resp.StatusCode < 300 && len(raw) > 0 {
		if err := json.Unmarshal(raw, out); err != nil {
			return resp.StatusCode, raw, fmt.Errorf("decoding %s %s response: %w", method, path, err)
		}
	}
	return resp.StatusCode, raw, nil
}
