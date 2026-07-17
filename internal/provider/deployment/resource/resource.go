package provider

import (
	"context"
	"net/http"
	"strings"
	"time"

	csv2 "github.com/computesphere/computesphere-go"
	cstypes "github.com/computesphere/terraform-provider-computesphere/internal/provider/types"
	"github.com/google/uuid"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

type DeploymentResource struct {
	client    *csv2.ClientWithResponses
	accountID string
}

var _ resource.Resource = &DeploymentResource{}
var _ resource.ResourceWithImportState = &DeploymentResource{}

func NewDeploymentResource() resource.Resource {
	return &DeploymentResource{}
}

func (r *DeploymentResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = "computesphere_deployment"
}

func (r *DeploymentResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = Schema(ctx)
}

type deploymentResourceModel struct {
	ID            types.String `tfsdk:"id"`
	Name          types.String `tfsdk:"name"`
	Status        types.String `tfsdk:"status"`
	ServiceID     types.String `tfsdk:"service_id"`
	EnvironmentID types.String `tfsdk:"environment_id"`
	ProjectID     types.String `tfsdk:"project_id"`
	Type          types.String `tfsdk:"type"`
	PlanID        types.String `tfsdk:"plan_id"`
	Image         types.String `tfsdk:"image"`
	ImageType     types.String `tfsdk:"image_type"`
	ImageProvider types.String `tfsdk:"image_provider"`
	ImageUsername types.String `tfsdk:"image_username"`
	ImagePassword types.String `tfsdk:"image_password"`
	ImageURL      types.String `tfsdk:"image_url"`
	BuildArgs     types.Map    `tfsdk:"build_args"`
	Port          types.Int64  `tfsdk:"port"`
	SphereCount   types.Int64  `tfsdk:"sphere_count"`
	EnvVars       types.Map    `tfsdk:"env_vars"`
	SecretVars    types.Map    `tfsdk:"secret_vars"`
	WaitForReady  types.Bool   `tfsdk:"wait_for_ready"`
}

func (r *DeploymentResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	data := cstypes.ConfigureResource(req, resp)
	if data != nil {
		r.client = data.V2Client
		r.accountID = data.AccountID
	}
}

func mapToStringPtr(ctx context.Context, m types.Map, diags *diag.Diagnostics) *map[string]string {
	if m.IsNull() || m.IsUnknown() {
		return nil
	}
	out := map[string]string{}
	diags.Append(m.ElementsAs(ctx, &out, false)...)
	return &out
}

func strPtr(s types.String) *string {
	if s.IsNull() || s.IsUnknown() || s.ValueString() == "" {
		return nil
	}
	v := s.ValueString()
	return &v
}

// buildUpdateBody assembles the DeploymentUpdateRequest from the model, wiring
// the container image (name + optional private-registry auth) and any public
// docker build args. Nil sub-objects are omitted so an update only touches the
// fields the practitioner actually set.
func buildUpdateBody(ctx context.Context, plan deploymentResourceModel, diags *diag.Diagnostics) csv2.DeploymentUpdateRequest {
	upd := csv2.DeploymentUpdateRequest{}

	img := csv2.DeploymentUpdateRequest_Image{}
	imgSet := false
	if name := strPtr(plan.Image); name != nil {
		img.Name = name
		imgSet = true
	}
	if t := strPtr(plan.ImageType); t != nil {
		it := csv2.DeploymentUpdateRequestImageType(*t)
		img.Type = &it
		imgSet = true
	}
	if p := strPtr(plan.ImageProvider); p != nil {
		img.Provider = p
		imgSet = true
	}
	if u := strPtr(plan.ImageUsername); u != nil {
		img.Username = u
		imgSet = true
	}
	if pw := strPtr(plan.ImagePassword); pw != nil {
		img.Password = pw
		imgSet = true
	}
	if url := strPtr(plan.ImageURL); url != nil {
		img.Url = url
		imgSet = true
	}
	if imgSet {
		upd.Image = &img
	}

	if args := mapToStringPtr(ctx, plan.BuildArgs, diags); args != nil {
		upd.BuildConfig = &csv2.DeploymentUpdateRequest_BuildConfig{BuildArgs: args}
	}

	if !plan.Port.IsNull() {
		p := int(plan.Port.ValueInt64())
		upd.Port = &p
	}
	return upd
}

func (r *DeploymentResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan deploymentResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	serviceID, err := uuid.Parse(plan.ServiceID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Invalid service_id", err.Error())
		return
	}
	envID, err := uuid.Parse(plan.EnvironmentID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Invalid environment_id", err.Error())
		return
	}
	projectID, err := uuid.Parse(plan.ProjectID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Invalid project_id", err.Error())
		return
	}

	body := csv2.CreateDeploymentRequest{
		Type:          plan.Type.ValueString(),
		ServiceId:     &serviceID,
		EnvironmentId: &envID,
		ProjectId:     &projectID,
	}
	if !plan.PlanID.IsNull() && plan.PlanID.ValueString() != "" {
		planID, perr := uuid.Parse(plan.PlanID.ValueString())
		if perr != nil {
			resp.Diagnostics.AddError("Invalid plan_id", perr.Error())
			return
		}
		body.PlanId = &planID
	}
	if !plan.Image.IsNull() && plan.Image.ValueString() != "" {
		body.Set("image", plan.Image.ValueString())
	}
	if !plan.Port.IsNull() {
		body.Set("port", int(plan.Port.ValueInt64()))
	}
	if !plan.SphereCount.IsNull() {
		body.Set("spherelets", int(plan.SphereCount.ValueInt64()))
	}

	createResp, err := r.client.CreateDeploymentWithResponse(ctx, body)
	if err != nil {
		resp.Diagnostics.AddError("Error creating deployment", err.Error())
		return
	}
	if createResp.StatusCode() != http.StatusCreated || createResp.JSON201 == nil {
		resp.Diagnostics.AddError("Error creating deployment", cstypes.ProblemSummary(createResp.Body, createResp.StatusCode()))
		return
	}
	depID, err := uuid.Parse(createResp.JSON201.Id)
	if err != nil {
		resp.Diagnostics.AddError("Error creating deployment", "the API returned an unparseable deployment id")
		return
	}
	plan.ID = types.StringValue(createResp.JSON201.Id)
	plan.Name = types.StringValue(createResp.JSON201.Name)

	// Apply env/secret vars before deploying.
	if envVars := mapToStringPtr(ctx, plan.EnvVars, &resp.Diagnostics); envVars != nil || !plan.SecretVars.IsNull() {
		secretVars := mapToStringPtr(ctx, plan.SecretVars, &resp.Diagnostics)
		if resp.Diagnostics.HasError() {
			return
		}
		if evResp, everr := r.client.ReplaceDeploymentEnvVarsWithResponse(ctx, depID, csv2.SetEnvVarsRequest{
			EnvVars:    envVars,
			SecretVars: secretVars,
		}); everr != nil {
			resp.Diagnostics.AddError("Error setting deployment env vars", everr.Error())
			return
		} else if evResp.StatusCode() != http.StatusOK {
			resp.Diagnostics.AddError("Error setting deployment env vars", cstypes.ProblemSummary(evResp.Body, evResp.StatusCode()))
			return
		}
	}

	// Apply private-registry auth and/or public build args via the update
	// surface before deploying; the create request only carries a flat image
	// name. Skipped unless one of these fields is set, so the common
	// public-image path stays a single create+deploy.
	if deploymentNeedsUpdate(ctx, plan, &resp.Diagnostics) {
		if resp.Diagnostics.HasError() {
			return
		}
		upd := buildUpdateBody(ctx, plan, &resp.Diagnostics)
		if resp.Diagnostics.HasError() {
			return
		}
		if uResp, uerr := r.client.UpdateDeploymentWithResponse(ctx, depID, upd); uerr != nil {
			resp.Diagnostics.AddError("Error updating deployment", uerr.Error())
			return
		} else if uResp.StatusCode() != http.StatusOK && uResp.StatusCode() != http.StatusAccepted {
			resp.Diagnostics.AddError("Error updating deployment", cstypes.ProblemSummary(uResp.Body, uResp.StatusCode()))
			return
		}
	}

	if !r.deploy(ctx, depID, plan, &resp.Diagnostics) {
		return
	}
	r.refreshStatus(ctx, depID, &plan, &resp.Diagnostics)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

// deploymentNeedsUpdate reports whether the model sets any field that must be
// applied through the update surface (private-image auth or public build args)
// rather than the create request.
func deploymentNeedsUpdate(ctx context.Context, plan deploymentResourceModel, diags *diag.Diagnostics) bool {
	if strPtr(plan.ImageType) != nil ||
		strPtr(plan.ImageProvider) != nil ||
		strPtr(plan.ImageUsername) != nil ||
		strPtr(plan.ImagePassword) != nil ||
		strPtr(plan.ImageURL) != nil {
		return true
	}
	return mapToStringPtr(ctx, plan.BuildArgs, diags) != nil
}

// deploy triggers a deploy and, when wait_for_ready is set, blocks until the
// deployment settles into a terminal state.
func (r *DeploymentResource) deploy(ctx context.Context, id uuid.UUID, plan deploymentResourceModel, diags *diag.Diagnostics) bool {
	deployBody := csv2.DeployDeploymentJSONRequestBody{}
	if !plan.SphereCount.IsNull() {
		deployBody["sphere_count"] = int(plan.SphereCount.ValueInt64())
	}
	dResp, err := r.client.DeployDeploymentWithResponse(ctx, id, deployBody)
	if err != nil {
		diags.AddError("Error deploying", err.Error())
		return false
	}
	if dResp.StatusCode() != http.StatusAccepted && dResp.StatusCode() != http.StatusOK {
		diags.AddError("Error deploying", cstypes.ProblemSummary(dResp.Body, dResp.StatusCode()))
		return false
	}
	if plan.WaitForReady.IsNull() || plan.WaitForReady.ValueBool() {
		return r.waitForReady(ctx, id, diags)
	}
	return true
}

func isTransient(s string) bool {
	s = strings.ToLower(s)
	for _, k := range []string{"deploy", "pending", "progress", "provision", "queue", "starting", "creating", "updating", "scaling"} {
		if strings.Contains(s, k) {
			return true
		}
	}
	return false
}

func isFailure(s string) bool {
	s = strings.ToLower(s)
	for _, k := range []string{"fail", "error", "crash", "cancel"} {
		if strings.Contains(s, k) {
			return true
		}
	}
	return false
}

func (r *DeploymentResource) waitForReady(ctx context.Context, id uuid.UUID, diags *diag.Diagnostics) bool {
	deadline := time.Now().Add(15 * time.Minute)
	for {
		st, err := r.client.GetDeploymentStatusWithResponse(ctx, id)
		if err != nil {
			diags.AddError("Error waiting for deployment", err.Error())
			return false
		}
		if st.StatusCode() == http.StatusOK && st.JSON200 != nil {
			s := st.JSON200.Status
			if isFailure(s) {
				diags.AddError("Deployment failed", "deployment reached status: "+s)
				return false
			}
			if !isTransient(s) {
				return true
			}
		}
		if time.Now().After(deadline) {
			diags.AddError("Timed out waiting for deployment", "the deployment did not become ready within 15m")
			return false
		}
		select {
		case <-ctx.Done():
			diags.AddError("Cancelled waiting for deployment", ctx.Err().Error())
			return false
		case <-time.After(10 * time.Second):
		}
	}
}

func (r *DeploymentResource) refreshStatus(ctx context.Context, id uuid.UUID, m *deploymentResourceModel, diags *diag.Diagnostics) {
	getResp, err := r.client.GetDeploymentWithResponse(ctx, id)
	if err != nil {
		diags.AddError("Error reading deployment", err.Error())
		return
	}
	if getResp.StatusCode() != http.StatusOK || getResp.JSON200 == nil {
		diags.AddError("Error reading deployment", cstypes.ProblemSummary(getResp.Body, getResp.StatusCode()))
		return
	}
	m.Name = types.StringValue(getResp.JSON200.Name)
	m.Status = types.StringValue(getResp.JSON200.Status)
	m.Type = types.StringValue(getResp.JSON200.Type)

	if st, serr := r.client.GetDeploymentStatusWithResponse(ctx, id); serr == nil && st.StatusCode() == http.StatusOK && st.JSON200 != nil {
		m.Status = types.StringValue(st.JSON200.Status)
	}
}

func (r *DeploymentResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state deploymentResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	id, err := uuid.Parse(state.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Invalid deployment id", err.Error())
		return
	}
	getResp, err := r.client.GetDeploymentWithResponse(ctx, id)
	if err != nil {
		resp.Diagnostics.AddError("Error reading deployment", err.Error())
		return
	}
	if getResp.StatusCode() == http.StatusNotFound {
		resp.State.RemoveResource(ctx)
		return
	}
	if getResp.StatusCode() != http.StatusOK || getResp.JSON200 == nil {
		resp.Diagnostics.AddError("Error reading deployment", cstypes.ProblemSummary(getResp.Body, getResp.StatusCode()))
		return
	}
	// Refresh the stable/computed fields; the config inputs stay authoritative.
	state.Name = types.StringValue(getResp.JSON200.Name)
	state.Status = types.StringValue(getResp.JSON200.Status)
	state.Type = types.StringValue(getResp.JSON200.Type)
	if st, serr := r.client.GetDeploymentStatusWithResponse(ctx, id); serr == nil && st.StatusCode() == http.StatusOK && st.JSON200 != nil {
		state.Status = types.StringValue(st.JSON200.Status)
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *DeploymentResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan, state deploymentResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	id, err := uuid.Parse(state.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Invalid deployment id", err.Error())
		return
	}
	plan.ID = state.ID

	upd := buildUpdateBody(ctx, plan, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}
	if uResp, uerr := r.client.UpdateDeploymentWithResponse(ctx, id, upd); uerr != nil {
		resp.Diagnostics.AddError("Error updating deployment", uerr.Error())
		return
	} else if uResp.StatusCode() != http.StatusOK && uResp.StatusCode() != http.StatusAccepted {
		resp.Diagnostics.AddError("Error updating deployment", cstypes.ProblemSummary(uResp.Body, uResp.StatusCode()))
		return
	}

	envVars := mapToStringPtr(ctx, plan.EnvVars, &resp.Diagnostics)
	secretVars := mapToStringPtr(ctx, plan.SecretVars, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}
	if envVars != nil || secretVars != nil {
		if evResp, everr := r.client.ReplaceDeploymentEnvVarsWithResponse(ctx, id, csv2.SetEnvVarsRequest{EnvVars: envVars, SecretVars: secretVars}); everr != nil {
			resp.Diagnostics.AddError("Error setting deployment env vars", everr.Error())
			return
		} else if evResp.StatusCode() != http.StatusOK {
			resp.Diagnostics.AddError("Error setting deployment env vars", cstypes.ProblemSummary(evResp.Body, evResp.StatusCode()))
			return
		}
	}

	if !r.deploy(ctx, id, plan, &resp.Diagnostics) {
		return
	}
	r.refreshStatus(ctx, id, &plan, &resp.Diagnostics)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *DeploymentResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state deploymentResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	id, err := uuid.Parse(state.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Invalid deployment id", err.Error())
		return
	}
	dResp, err := r.client.DeleteDeploymentWithResponse(ctx, id)
	if err != nil {
		resp.Diagnostics.AddError("Error deleting deployment", err.Error())
		return
	}
	switch dResp.StatusCode() {
	case http.StatusOK, http.StatusNoContent, http.StatusAccepted, http.StatusNotFound:
		resp.State.RemoveResource(ctx)
	default:
		resp.Diagnostics.AddError("Error deleting deployment", cstypes.ProblemSummary(dResp.Body, dResp.StatusCode()))
	}
}

func (r *DeploymentResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}
