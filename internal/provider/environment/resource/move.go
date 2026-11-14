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
	"github.com/google/uuid"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var _ resource.ResourceWithModifyPlan = &EnvironmentResource{}

// ModifyPlan refuses a region change on an existing environment. The API
// can't move an environment to another region, and replacing it would delete
// everything in it, so the plan stops instead. It also refuses a project
// change while the environment runs services: a move takes its services to
// the new project (moving, copying or deleting them), which a configuration
// that manages them would then plan to replace.
func (r *EnvironmentResource) ModifyPlan(ctx context.Context, req resource.ModifyPlanRequest, resp *resource.ModifyPlanResponse) {
	if req.State.Raw.IsNull() || req.Plan.Raw.IsNull() {
		return
	}
	var planProject, stateProject, id types.String
	resp.Diagnostics.Append(req.Plan.GetAttribute(ctx, path.Root("project_id"), &planProject)...)
	resp.Diagnostics.Append(req.State.GetAttribute(ctx, path.Root("project_id"), &stateProject)...)
	resp.Diagnostics.Append(req.State.GetAttribute(ctx, path.Root("id"), &id)...)
	if resp.Diagnostics.HasError() {
		return
	}
	if planProject.IsUnknown() && r.client != nil {
		// The new project is only known at apply (e.g. a project created in
		// the same run): Update checks again then. Say so now.
		if n, err := r.deploymentCount(ctx, id.ValueString()); err == nil && n > 0 {
			resp.Diagnostics.AddAttributeWarning(path.Root("project_id"), "project_id may change at apply",
				fmt.Sprintf("project_id is only known at apply. This environment runs %d service(s), so if the "+
					"project changes, the apply stops before anything is moved.", n))
		}
	}
	if !planProject.IsUnknown() && !planProject.IsNull() && !planProject.Equal(stateProject) && r.client != nil {
		n, err := r.deploymentCount(ctx, id.ValueString())
		switch {
		case err != nil:
			resp.Diagnostics.AddAttributeError(path.Root("project_id"), "Can't check the environment's services", err.Error())
			return
		case n > 0:
			resp.Diagnostics.AddAttributeError(path.Root("project_id"), "project_id can't change while the environment runs services",
				fmt.Sprintf("This environment runs %d service(s). Moving it to another project takes its services "+
					"with it, and the computesphere_service resources that manage them would then be replaced. Move "+
					"the environment in the console or with the CLI, then set project_id here and on the services that "+
					"moved with it (a service that also runs in other environments is copied, and its resource stays as it is).", n))
			return
		default:
			resp.Diagnostics.AddAttributeWarning(path.Root("project_id"), "The environment moves to another project",
				"Who can read its variables and secrets follows the new project's members.")
		}
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

// deploymentCount is how many deployments (services) the environment runs.
func (r *EnvironmentResource) deploymentCount(ctx context.Context, eid string) (int, error) {
	id, err := uuid.Parse(eid)
	if err != nil {
		return 0, err
	}
	size := 1
	list, err := r.client.ListDeploymentsWithResponse(ctx, &csv2.ListDeploymentsParams{EnvironmentId: &id, PageSize: &size})
	if err != nil {
		return 0, err
	}
	if list.StatusCode() != http.StatusOK || list.JSON200 == nil {
		return 0, fmt.Errorf("%s", cstypes.ProblemSummary(list.Body, list.StatusCode()))
	}
	if list.JSON200.Pagination.Total > 0 {
		return list.JSON200.Pagination.Total, nil
	}
	return len(list.JSON200.Items), nil
}
