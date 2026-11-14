package provider

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
	cstypes "github.com/computesphere/terraform-provider-computesphere/internal/provider/types"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var _ resource.ResourceWithModifyPlan = &EnvironmentResource{}

// ModifyPlan refuses a region change on an existing environment. The API
// can't move an environment to another region, and replacing it would delete
// everything in it, so the plan stops instead.
func (r *EnvironmentResource) ModifyPlan(ctx context.Context, req resource.ModifyPlanRequest, resp *resource.ModifyPlanResponse) {
	if req.State.Raw.IsNull() || req.Plan.Raw.IsNull() {
		return
	}
	var planRegion, stateRegion types.String
	resp.Diagnostics.Append(req.Plan.GetAttribute(ctx, path.Root("region"), &planRegion)...)
	resp.Diagnostics.Append(req.State.GetAttribute(ctx, path.Root("region"), &stateRegion)...)
	if resp.Diagnostics.HasError() || planRegion.IsUnknown() || planRegion.IsNull() || planRegion.Equal(stateRegion) {
		return
	}
	resp.Diagnostics.AddAttributeError(path.Root("region"), "region can't change on an existing environment",
		fmt.Sprintf("This environment is in %s, and an environment can't move to another region. Create a new "+
			"environment in %s as a separate resource, move your services to it, then remove this one. To keep "+
			"this environment, set region back to %q.", stateRegion.ValueString(), planRegion.ValueString(), stateRegion.ValueString()))
}

// moveEnvironment calls POST /environments/{eid}/move, which moves the
// environment and everything in it to another project in the same account.
// The SDK in use doesn't have the operation, so the request goes through the
// SDK client's base URL, auth and account headers.
func moveEnvironment(ctx context.Context, client *csv2.ClientWithResponses, eid, targetProjectID string) error {
	c, ok := client.ClientInterface.(*csv2.Client)
	if !ok {
		return fmt.Errorf("unexpected API client type %T", client.ClientInterface)
	}
	base, err := url.Parse(c.Server)
	if err != nil {
		return err
	}
	if !strings.HasSuffix(base.Path, "/") {
		base.Path += "/"
	}
	u := base.ResolveReference(&url.URL{Path: "environments/" + url.PathEscape(eid) + "/move"})
	body, err := json.Marshal(map[string]string{"target_project_id": targetProjectID})
	if err != nil {
		return err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, u.String(), bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	for _, edit := range c.RequestEditors {
		if err := edit(ctx, req); err != nil {
			return err
		}
	}
	resp, err := c.Client.Do(req)
	if err != nil {
		return err
	}
	defer func() { _ = resp.Body.Close() }()
	raw, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("%s", cstypes.ProblemSummary(raw, resp.StatusCode))
	}
	return nil
}
