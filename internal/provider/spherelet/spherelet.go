// Package spherelet holds the spherelet type wire contract shared by the
// service and deployment resources and the computesphere_spherelet_types data
// source: the API field names, the list endpoint, the list item shape and the
// alias rules. Every name the API defines for spherelet types lives in this
// file, so a contract change is a one-file edit.
package spherelet

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"

	csv2 "github.com/computesphere/computesphere-go"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

const (
	// FieldType is the request and response field carrying the spherelet type code.
	FieldType = "spherelet_type"
	// FieldPlanID is the deprecated request and response field it replaces.
	FieldPlanID = "plan_id"

	// ListPath is the list endpoint (GET /v2/spherelet-types), relative to
	// the v2 base URL. Tests route their fake API with it too.
	ListPath = "spherelet-types"

	// fieldRaw is the deployment read's nested record, which carries the
	// spherelet type of a deployment.
	fieldRaw = "raw"
)

// ListQuery is the list endpoint's query string: the optional region.
func ListQuery(region string) url.Values {
	q := url.Values{}
	if region != "" {
		q.Set("region", region)
	}
	return q
}

// Item is one entry of GET /v2/spherelet-types.
type Item struct {
	ID                string   `json:"id"`
	Name              string   `json:"name"`
	Family            string   `json:"family"`
	VCPU              float64  `json:"vcpu"`
	CPUGuaranteedVCPU float64  `json:"cpu_guaranteed_vcpu"`
	MemoryMB          int64    `json:"memory_mb"`
	MonthlyPriceUSD   float64  `json:"monthly_price_usd"`
	AnnualPriceUSD    float64  `json:"annual_price_usd"`
	Available         bool     `json:"available"`
	Aliases           []string `json:"aliases"`
}

type listResponse struct {
	Object string `json:"object"`
	Items  []Item `json:"items"`
}

// staticAliases maps the original size names to their codes. The list
// endpoint's `aliases` field is authoritative; this map is only the fallback
// used when the list can't be fetched.
var staticAliases = map[string]string{
	"FLX": "sl-0.25c-512m",
	"MAX": "sl-1c-2g",
	"PWR": "sl-2c-4g",
}

// List calls GET /v2/spherelet-types, optionally for one region. The SDK
// client is reused for its base URL, auth and account headers.
func List(ctx context.Context, client *csv2.ClientWithResponses, region string) ([]Item, int, []byte, error) {
	if client == nil {
		return nil, 0, nil, fmt.Errorf("the provider is not configured")
	}
	c, ok := client.ClientInterface.(*csv2.Client)
	if !ok {
		return nil, 0, nil, fmt.Errorf("unexpected API client type %T", client.ClientInterface)
	}
	base, err := url.Parse(c.Server)
	if err != nil {
		return nil, 0, nil, err
	}
	if !strings.HasSuffix(base.Path, "/") {
		base.Path += "/"
	}
	u := base.ResolveReference(&url.URL{Path: ListPath})
	u.RawQuery = ListQuery(region).Encode()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil)
	if err != nil {
		return nil, 0, nil, err
	}
	for _, edit := range c.RequestEditors {
		if err := edit(ctx, req); err != nil {
			return nil, 0, nil, err
		}
	}
	resp, err := c.Client.Do(req)
	if err != nil {
		return nil, 0, nil, err
	}
	defer func() { _ = resp.Body.Close() }()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, resp.StatusCode, nil, err
	}
	if resp.StatusCode != http.StatusOK {
		return nil, resp.StatusCode, body, nil
	}
	var out listResponse
	if err := json.Unmarshal(body, &out); err != nil {
		return nil, resp.StatusCode, body, fmt.Errorf("could not read the spherelet size list: %w", err)
	}
	return out.Items, resp.StatusCode, body, nil
}

// Fields are the spherelet type fields of a service or deployment response.
// A nil pointer means the field was absent or null.
type Fields struct {
	SphereletType *string
	PlanID        *string
	// Present reports which of the two keys the response carried at all.
	SphereletTypePresent bool
	PlanIDPresent        bool
}

// ReadFields extracts spherelet_type and plan_id from a service response body.
func ReadFields(body []byte) Fields {
	var raw map[string]json.RawMessage
	if err := json.Unmarshal(body, &raw); err != nil {
		return Fields{}
	}
	return readFields(raw)
}

// ReadDeploymentFields extracts spherelet_type and plan_id from a deployment
// response body, where they sit in the nested `raw` record.
func ReadDeploymentFields(body []byte) Fields {
	var outer map[string]json.RawMessage
	if err := json.Unmarshal(body, &outer); err != nil {
		return Fields{}
	}
	var raw map[string]json.RawMessage
	if err := json.Unmarshal(outer[fieldRaw], &raw); err != nil {
		return Fields{}
	}
	return readFields(raw)
}

func readFields(raw map[string]json.RawMessage) Fields {
	var f Fields
	read := func(key string) (*string, bool) {
		v, ok := raw[key]
		if !ok {
			return nil, false
		}
		var s *string
		_ = json.Unmarshal(v, &s)
		if s != nil && *s == "" {
			s = nil
		}
		return s, true
	}
	f.SphereletType, f.SphereletTypePresent = read(FieldType)
	f.PlanID, f.PlanIDPresent = read(FieldPlanID)
	return f
}

// SetFields marshals body (an SDK request struct) and adds the given extra
// fields, skipping empty ones. It returns the JSON request body.
func SetFields(body any, extra map[string]string) ([]byte, error) {
	b, err := json.Marshal(body)
	if err != nil {
		return nil, err
	}
	var m map[string]any
	if err := json.Unmarshal(b, &m); err != nil {
		return nil, err
	}
	for k, v := range extra {
		if v != "" {
			m[k] = v
		}
	}
	return json.Marshal(m)
}

// IsAlias reports whether configured names the same spherelet type as code:
// either the same code, or one of the code's aliases (any case). items is the
// list from the API when available; when it is missing or doesn't list code,
// the static map is used.
func IsAlias(configured, code string, items []Item) bool {
	if configured == code {
		return true
	}
	for _, it := range items {
		if it.ID != code {
			continue
		}
		for _, a := range it.Aliases {
			if strings.EqualFold(a, configured) {
				return true
			}
		}
		return false
	}
	return staticAliases[strings.ToUpper(configured)] == code
}

// Value picks the state value for an attribute the API returns. prior is the
// planned or prior-state value, present/returned what the response carried.
//
//   - Absent from the response: keep a known prior value, else null.
//   - Returned: the API value, except that a configured alias the API resolved
//     to its code is kept as written, so an alias in config never shows a diff.
//     aliasItems is called only in that case.
func Value(prior types.String, returned *string, present bool, aliasItems func() []Item) types.String {
	if !present {
		if prior.IsUnknown() {
			return types.StringNull()
		}
		return prior
	}
	if returned == nil {
		return types.StringNull()
	}
	if !prior.IsNull() && !prior.IsUnknown() && prior.ValueString() != *returned && aliasItems != nil &&
		IsAlias(prior.ValueString(), *returned, aliasItems()) {
		return prior
	}
	return types.StringValue(*returned)
}

// AliasLookup returns a function that fetches the spherelet type list once,
// on first call, for alias checks. A failed fetch yields nil, which makes
// IsAlias fall back to the static map.
func AliasLookup(ctx context.Context, client *csv2.ClientWithResponses) func() []Item {
	var (
		items   []Item
		fetched bool
	)
	return func() []Item {
		if !fetched {
			fetched = true
			if got, status, _, err := List(ctx, client, ""); err == nil && status == http.StatusOK {
				items = got
			}
		}
		return items
	}
}

// ConflictDetail is the plan-time error shown when both spherelet_type and the
// deprecated plan_id are set.
const ConflictDetail = "Set spherelet_type or the deprecated plan_id, not both."
