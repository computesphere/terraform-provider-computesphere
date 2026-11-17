package provider

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	csv2 "github.com/computesphere/computesphere-go"
	"github.com/computesphere/terraform-provider-computesphere/internal/provider/spherelet"
	cstypes "github.com/computesphere/terraform-provider-computesphere/internal/provider/types"
	"github.com/google/uuid"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// waitTimeout bounds how long a create or change waits for the deploy.
const waitTimeout = 15 * time.Minute

// pollInterval is how often the deployment status is read while waiting.
var pollInterval = 10 * time.Second

// SetPollIntervalForTest shortens the status poll in tests. Test use only.
func SetPollIntervalForTest(d time.Duration) { pollInterval = d }

// computeResource is one compute kind's resource.
type computeResource struct {
	kind   kind
	client *csv2.ClientWithResponses
}

var _ resource.Resource = &computeResource{}
var _ resource.ResourceWithImportState = &computeResource{}
var _ resource.ResourceWithMoveState = &computeResource{}
var _ resource.ResourceWithModifyPlan = &computeResource{}
var _ resource.ResourceWithValidateConfig = &computeResource{}

func NewWebServiceResource() resource.Resource     { return &computeResource{kind: webService} }
func NewPrivateServiceResource() resource.Resource { return &computeResource{kind: privateService} }
func NewWorkerResource() resource.Resource         { return &computeResource{kind: worker} }
func NewCronJobResource() resource.Resource        { return &computeResource{kind: cronJob} }
func NewStaticSiteResource() resource.Resource     { return &computeResource{kind: staticSite} }

func (r *computeResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = r.kind.typeName
}

func (r *computeResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = Schema(r.kind)
}

func (r *computeResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if data := cstypes.ConfigureResource(req, resp); data != nil {
		r.client = data.V2Client
	}
}

// getModel reads a plan or state into the superset model.
func getModel(ctx context.Context, get func(context.Context, any) diag.Diagnostics, diags *diag.Diagnostics) model {
	var obj types.Object
	diags.Append(get(ctx, &obj)...)
	if diags.HasError() {
		return model{}
	}
	m, d := fromObject(ctx, obj)
	diags.Append(d...)
	return m
}

// withConfigPassword copies the write-only image_password from the
// configuration into m; plan and state never hold it.
func (r *computeResource) withConfigPassword(ctx context.Context, config tfsdk.Config, m *model, diags *diag.Diagnostics) {
	if !r.kind.has("image_password") {
		return
	}
	var pw types.String
	diags.Append(config.GetAttribute(ctx, path.Root("image_password"), &pw)...)
	m.ImagePassword = pw
}

// attrGetter is what tfsdk.Plan and tfsdk.State share.
type attrGetter interface {
	GetAttribute(ctx context.Context, p path.Path, target interface{}) diag.Diagnostics
}

// ValidateConfig refuses a worker that sets both command (an image
// worker's command) and git_url: a git worker runs start_command.
func (r *computeResource) ValidateConfig(ctx context.Context, req resource.ValidateConfigRequest, resp *resource.ValidateConfigResponse) {
	r.validateServiceOrName(ctx, req.Config, &resp.Diagnostics)
	if !r.kind.is(worker) {
		return
	}
	var command, gitURL types.String
	resp.Diagnostics.Append(req.Config.GetAttribute(ctx, path.Root("command"), &command)...)
	resp.Diagnostics.Append(req.Config.GetAttribute(ctx, path.Root("git_url"), &gitURL)...)
	if resp.Diagnostics.HasError() || command.IsNull() || gitURL.IsNull() {
		return
	}
	resp.Diagnostics.AddAttributeError(path.Root("command"), "command and git_url can't both be set",
		"command is the command an image worker runs. A worker built from git_url runs start_command instead: "+
			"set start_command and remove command, or remove git_url to run an image.")
}

// validateServiceOrName asks for exactly one of service_id (an existing
// service) and name (a new service created with this one).
func (r *computeResource) validateServiceOrName(ctx context.Context, config tfsdk.Config, diags *diag.Diagnostics) {
	var serviceID, name types.String
	diags.Append(config.GetAttribute(ctx, path.Root("service_id"), &serviceID)...)
	diags.Append(config.GetAttribute(ctx, path.Root("name"), &name)...)
	if diags.HasError() || serviceID.IsUnknown() || name.IsUnknown() {
		return
	}
	switch {
	case serviceID.IsNull() && name.IsNull():
		diags.AddAttributeError(path.Root("name"), "Set service_id or name",
			fmt.Sprintf("Set service_id to run an existing service, or name to create a new service with this %s.",
				r.kind.noun))
	case !serviceID.IsNull() && !name.IsNull():
		diags.AddAttributeError(path.Root("name"), "Set service_id or name, not both",
			"With service_id, the name is the service's. Remove name, or remove service_id to create a new service.")
	}
}

// ModifyPlan refuses a git_repo_id that differs from the one the service
// has, which the API can't change, instead of replacing a running service
// for it. A first value over null in state is only recorded.
func (r *computeResource) ModifyPlan(ctx context.Context, req resource.ModifyPlanRequest, resp *resource.ModifyPlanResponse) {
	if req.Plan.Raw.IsNull() {
		return
	}
	if req.State.Raw.IsNull() {
		r.refuseDefaultURLOffAtCreate(ctx, req.Plan, &resp.Diagnostics)
		return
	}
	r.refuseGitRepoIDChange(ctx, req.Plan, req.State, &resp.Diagnostics)
	r.refuseGitToImage(ctx, req.Plan, req.State, &resp.Diagnostics)
}

// refuseDefaultURLOffAtCreate refuses default_url_disabled = true on a new
// service: the API turns the default address off only once a custom domain
// is verified and serving, which a service that doesn't exist yet can't have.
func (r *computeResource) refuseDefaultURLOffAtCreate(ctx context.Context, plan attrGetter, diags *diag.Diagnostics) {
	if !r.kind.has("network_access") {
		return
	}
	p := path.Root("network_access").AtName("default_url_disabled")
	var off types.Bool
	diags.Append(plan.GetAttribute(ctx, p, &off)...)
	if diags.HasError() || !off.ValueBool() {
		return
	}
	diags.AddAttributeError(p, "Can't turn the default address off on a new "+r.kind.noun,
		fmt.Sprintf("The default address can be turned off only once the %s has a custom domain that is verified "+
			"and serving. Create the %s without default_url_disabled, add its custom domain "+
			"(computesphere_custom_domain), then set default_url_disabled = true.", r.kind.noun, r.kind.noun))
}

// refuseGitToImage refuses switching a git-built service to an image in
// place. The API applies a change body's image before its build settings
// and doesn't check the second before writing the first, so the switch
// could be left half-applied. At plan time an image that is still unknown
// counts when git_url is gone or unknown, since it could become a switch;
// Update checks again with the known values.
func (r *computeResource) refuseGitToImage(ctx context.Context, plan, state attrGetter, diags *diag.Diagnostics) {
	if !r.kind.has("image") || !r.kind.has("git_url") {
		return
	}
	var planGit, stateGit, planImage types.String
	diags.Append(plan.GetAttribute(ctx, path.Root("git_url"), &planGit)...)
	diags.Append(state.GetAttribute(ctx, path.Root("git_url"), &stateGit)...)
	diags.Append(plan.GetAttribute(ctx, path.Root("image"), &planImage)...)
	if diags.HasError() {
		return
	}
	set := func(v types.String) bool { return !v.IsNull() && !v.IsUnknown() && v.ValueString() != "" }
	mayBeImage := set(planImage) || planImage.IsUnknown()
	if !set(stateGit) || set(planGit) || !mayBeImage {
		return
	}
	diags.AddAttributeError(path.Root("image"), "Can't switch from git to an image in place",
		fmt.Sprintf("This %s builds from git, and the API can't switch it to an image in one change yet. To run "+
			"an image, create a new %s with it (under another resource name), move traffic or work over, then "+
			"remove this one. Or keep git_url and leave image out.", r.kind.noun, r.kind.noun))
}

// refuseGitRepoIDChange refuses a git_repo_id that differs from a recorded
// one: no change body takes it. Null in state (after an import or a move)
// isn't a change; the value is recorded without an API call.
func (r *computeResource) refuseGitRepoIDChange(ctx context.Context, plan, state attrGetter, diags *diag.Diagnostics) {
	if !r.kind.has("git_repo_id") {
		return
	}
	var planV, stateV types.String
	diags.Append(plan.GetAttribute(ctx, path.Root("git_repo_id"), &planV)...)
	diags.Append(state.GetAttribute(ctx, path.Root("git_repo_id"), &stateV)...)
	if diags.HasError() || planV.IsUnknown() || planV.IsNull() || stateV.IsNull() || stateV.ValueString() == "" || planV.Equal(stateV) {
		return
	}
	diags.AddAttributeError(path.Root("git_repo_id"), "git_repo_id can't change on an existing "+r.kind.noun,
		fmt.Sprintf("This %s has git_repo_id %q, and the API can't change it in place. This provider "+
			"won't replace a running %s for it. Set git_repo_id back to %q.",
			r.kind.noun, stateV.ValueString(), r.kind.noun, stateV.ValueString()))
}

// setModel writes the model to state as this kind's object. The write-only
// image_password is always null there.
func (r *computeResource) setModel(ctx context.Context, set func(context.Context, any) diag.Diagnostics, m model, diags *diag.Diagnostics) {
	m.ImagePassword = types.StringNull()
	obj, d := toObject(ctx, r.kind, m)
	diags.Append(d...)
	if diags.HasError() {
		return
	}
	diags.Append(set(ctx, obj)...)
}

// apiWarnings shows the `warnings` of a create or change response as
// Terraform warnings: a field the API didn't use, a size it chose
// (instance_type_fallback), a variable that looks like a credential. None
// of them refused the request.
func apiWarnings(body []byte, what string, diags *diag.Diagnostics) {
	var out struct {
		Warnings []struct {
			Code    string `json:"code"`
			Field   string `json:"field"`
			Message string `json:"message"`
		} `json:"warnings"`
	}
	if json.Unmarshal(body, &out) != nil {
		return
	}
	for _, w := range out.Warnings {
		summary := "The API warned about " + what
		if w.Field != "" {
			summary += ": " + w.Field
		}
		diags.AddWarning(summary, fmt.Sprintf("%s (%s)", w.Message, w.Code))
	}
}

func jsonBody(v any) (*bytes.Reader, error) {
	b, err := json.Marshal(v)
	if err != nil {
		return nil, err
	}
	return bytes.NewReader(b), nil
}

func (r *computeResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	plan := getModel(ctx, req.Plan.Get, &resp.Diagnostics)
	r.withConfigPassword(ctx, req.Config, &plan, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}
	for _, f := range []struct {
		attr string
		v    types.String
	}{{"service_id", plan.ServiceID}, {"environment_id", plan.EnvironmentID}, {"project_id", plan.ProjectID}} {
		if f.attr == "service_id" && !present(f.v) {
			continue // a standalone create names a new service instead
		}
		if _, err := uuid.Parse(f.v.ValueString()); err != nil {
			resp.Diagnostics.AddAttributeError(path.Root(f.attr), "Invalid "+f.attr, err.Error())
		}
	}
	if resp.Diagnostics.HasError() {
		return
	}

	body := createBody(ctx, r.kind, plan, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}
	rdr, err := jsonBody(body)
	if err != nil {
		resp.Diagnostics.AddError("Error creating "+r.kind.noun, err.Error())
		return
	}
	created, err := r.client.CreateDeploymentWithBodyWithResponse(ctx, "application/json", rdr)
	if err != nil {
		resp.Diagnostics.AddError("Error creating "+r.kind.noun, err.Error())
		return
	}
	if created.StatusCode() != http.StatusCreated || created.JSON201 == nil {
		resp.Diagnostics.AddError("Error creating "+r.kind.noun, cstypes.ProblemSummary(created.Body, created.StatusCode()))
		return
	}
	apiWarnings(created.Body, "the new "+r.kind.noun, &resp.Diagnostics)
	id, err := uuid.Parse(created.JSON201.Id)
	if err != nil {
		resp.Diagnostics.AddError("Error creating "+r.kind.noun, "the API returned an unparseable deployment id")
		return
	}
	state := plan
	if !present(state.ServiceID) {
		// The API created the service; take its id from the new deployment.
		state.ServiceID = types.StringNull()
		if created.JSON201.Raw != nil {
			if v, ok := created.JSON201.Raw.AdditionalProperties["service_id"].(string); ok && v != "" {
				state.ServiceID = types.StringValue(v)
			}
		}
	}
	state.ID = types.StringValue(created.JSON201.Id)
	state.Name = types.StringValue(created.JSON201.Name)
	state.Status = types.StringValue(created.JSON201.Status)
	applyOutputs(&state, created.JSON201.Outputs, r.kind)
	settleUnknowns(&state)
	// Save the id first so a failed step below doesn't orphan the deployment.
	r.setModel(ctx, resp.State.Set, state, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}

	if extra := postCreateBody(plan); len(extra) > 0 {
		if !r.patch(ctx, id, extra, &resp.Diagnostics) {
			return
		}
	}
	// A git-built create starts its first build, which rolls the service out;
	// a /deploy before it finishes is refused. The nested form (git_repo_id)
	// starts none, so start it here.
	switch {
	case nestedGit(r.kind, plan):
		if !r.build(ctx, id, &resp.Diagnostics) {
			return
		}
	case !gitBuilt(r.kind, plan):
		if !r.deploy(ctx, id, &resp.Diagnostics) {
			return
		}
	}
	if waitFor(plan) && !r.waitForReady(ctx, id, &resp.Diagnostics) {
		return
	}
	r.refresh(ctx, id, &state, &resp.Diagnostics)
	r.setModel(ctx, resp.State.Set, state, &resp.Diagnostics)
}

// settleUnknowns nulls computed values still unknown, so a state saved
// part-way through create holds no unknowns.
func settleUnknowns(m *model) {
	if m.URL.IsUnknown() {
		m.URL = types.StringNull()
	}
	if m.InternalURL.IsUnknown() {
		m.InternalURL = types.StringNull()
	}
	if m.Status.IsUnknown() {
		m.Status = types.StringNull()
	}
	if m.Name.IsUnknown() {
		m.Name = types.StringNull()
	}
	if m.GitRepoID.IsUnknown() {
		m.GitRepoID = types.StringNull()
	}
	if m.InstanceType.IsUnknown() {
		m.InstanceType = types.StringNull()
	}
}

func waitFor(m model) bool { return m.WaitForReady.IsNull() || m.WaitForReady.ValueBool() }

// patch sends a change body with the kind's type discriminator. It never
// rolls out on its own: the one deploy at the end of Create and Update rolls
// out every saved change together.
func (r *computeResource) patch(ctx context.Context, id uuid.UUID, body map[string]any, diags *diag.Diagnostics) bool {
	body["type"] = r.kind.apiType
	body["skip_redeploy"] = true
	rdr, err := jsonBody(body)
	if err != nil {
		diags.AddError("Error updating "+r.kind.noun, err.Error())
		return false
	}
	resp, err := r.client.UpdateDeploymentWithBodyWithResponse(ctx, id, "application/json", rdr)
	if err != nil {
		diags.AddError("Error updating "+r.kind.noun, err.Error())
		return false
	}
	if resp.StatusCode() != http.StatusOK && resp.StatusCode() != http.StatusAccepted {
		diags.AddError("Error updating "+r.kind.noun, cstypes.ProblemSummary(resp.Body, resp.StatusCode()))
		return false
	}
	apiWarnings(resp.Body, "the "+r.kind.noun+"'s change", diags)
	return true
}

func (r *computeResource) deploy(ctx context.Context, id uuid.UUID, diags *diag.Diagnostics) bool {
	resp, err := r.client.DeployDeploymentWithResponse(ctx, id, csv2.DeployDeploymentJSONRequestBody{})
	if err != nil {
		diags.AddError("Error deploying "+r.kind.noun, err.Error())
		return false
	}
	if resp.StatusCode() != http.StatusAccepted && resp.StatusCode() != http.StatusOK {
		diags.AddError("Error deploying "+r.kind.noun, cstypes.ProblemSummary(resp.Body, resp.StatusCode()))
		return false
	}
	return true
}

// build starts a build of the saved source and settings
// (POST /deployments/{id}/builds with an empty manifest, which the API
// completes from the saved one); a successful build rolls the service out.
func (r *computeResource) build(ctx context.Context, id uuid.UUID, diags *diag.Diagnostics) bool {
	body := map[string]any{"manifest": map[string]any{}}
	status, raw, err := cstypes.DoJSON(ctx, r.client, http.MethodPost, "/deployments/"+id.String()+"/builds", nil, body, nil)
	if err != nil {
		diags.AddError("Error building "+r.kind.noun, err.Error())
		return false
	}
	if status != http.StatusOK && status != http.StatusAccepted && status != http.StatusCreated {
		diags.AddError("Error building "+r.kind.noun, cstypes.ProblemSummary(raw, status))
		return false
	}
	return true
}

// setEnvVars writes the variable maps that changed, each as the whole set
// (replace=true): a map left out is kept as it is, and an empty one clears
// it, so a key or map removed from the configuration is removed.
func (r *computeResource) setEnvVars(ctx context.Context, id uuid.UUID, plan, state model, diags *diag.Diagnostics) bool {
	skip := true
	req := csv2.SetEnvVarsRequest{SkipRedeploy: &skip}
	if !plan.EnvVars.Equal(state.EnvVars) {
		req.EnvVars = cstypes.MapOrEmpty(ctx, plan.EnvVars, diags)
	}
	if !plan.SecretVars.Equal(state.SecretVars) {
		req.SecretVars = cstypes.MapOrEmpty(ctx, plan.SecretVars, diags)
	}
	if diags.HasError() {
		return false
	}
	resp, err := r.client.ReplaceDeploymentEnvVarsWithResponse(ctx, id, req, cstypes.WholeSet)
	if err != nil {
		diags.AddError("Error setting environment variables", err.Error())
		return false
	}
	if resp.StatusCode() != http.StatusOK {
		diags.AddError("Error setting environment variables", cstypes.ProblemSummary(resp.Body, resp.StatusCode()))
		return false
	}
	apiWarnings(resp.Body, "the "+r.kind.noun+"'s variables", diags)
	return true
}

// scale changes the spherelet count, the size, or both, through
// POST /deployments/{id}/scale, sending only what changed. The SDK's request
// type always sends spherelets and has no instance_type, so the body is raw.
func (r *computeResource) scale(ctx context.Context, id uuid.UUID, count types.Int64, size types.String, diags *diag.Diagnostics) bool {
	body := map[string]any{"skip_redeploy": true}
	if present(count) {
		body["spherelets"] = count.ValueInt64()
	}
	if present(size) && size.ValueString() != "" {
		body[spherelet.FieldType] = size.ValueString()
	}
	status, raw, err := cstypes.DoJSON(ctx, r.client, http.MethodPost, "/deployments/"+id.String()+"/scale", nil, body, nil)
	if err != nil {
		diags.AddError("Error scaling "+r.kind.noun, err.Error())
		return false
	}
	if status != http.StatusOK && status != http.StatusAccepted {
		diags.AddError("Error scaling "+r.kind.noun, cstypes.ProblemSummary(raw, status))
		return false
	}
	return true
}

func isTransient(s string) bool {
	s = strings.ToLower(s)
	for _, k := range []string{"deploy", "pending", "progress", "provision", "queue", "starting", "creating", "updating", "scaling", "build"} {
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

func (r *computeResource) waitForReady(ctx context.Context, id uuid.UUID, diags *diag.Diagnostics) bool {
	deadline := time.Now().Add(waitTimeout)
	for {
		st, err := r.client.GetDeploymentStatusWithResponse(ctx, id)
		if err != nil {
			diags.AddError("Error waiting for "+r.kind.noun, err.Error())
			return false
		}
		if st.StatusCode() == http.StatusOK && st.JSON200 != nil {
			s := st.JSON200.Status
			if isFailure(s) {
				diags.AddError("The "+r.kind.noun+" failed to deploy", "status: "+s)
				return false
			}
			if !isTransient(s) {
				return true
			}
		}
		if time.Now().After(deadline) {
			diags.AddError("Timed out waiting for "+r.kind.noun,
				fmt.Sprintf("the %s didn't finish deploying within %s", r.kind.noun, waitTimeout))
			return false
		}
		select {
		case <-ctx.Done():
			diags.AddError("Cancelled waiting for "+r.kind.noun, ctx.Err().Error())
			return false
		case <-time.After(pollInterval):
		}
	}
}

func applyOutputs(m *model, outputs *map[string]string, k kind) {
	if outputs == nil {
		return
	}
	if k.has("url") {
		if v, ok := (*outputs)["url"]; ok {
			m.URL = types.StringValue(v)
		}
	}
	if k.has("internal_url") {
		if v, ok := (*outputs)["internal_url"]; ok {
			m.InternalURL = types.StringValue(v)
		}
	}
}

// refresh reads the computed fields: name, status and addresses. The
// configured fields stay authoritative.
func (r *computeResource) refresh(ctx context.Context, id uuid.UUID, m *model, diags *diag.Diagnostics) bool {
	got, err := r.client.GetDeploymentWithResponse(ctx, id)
	if err != nil {
		diags.AddError("Error reading "+r.kind.noun, err.Error())
		return false
	}
	if got.StatusCode() == http.StatusNotFound {
		return false
	}
	if got.StatusCode() != http.StatusOK || got.JSON200 == nil {
		diags.AddError("Error reading "+r.kind.noun, cstypes.ProblemSummary(got.Body, got.StatusCode()))
		return false
	}
	d := got.JSON200
	if !r.kind.matches(d.Type) {
		diags.AddError("Wrong resource type for this deployment",
			fmt.Sprintf("Deployment %s is a %s, not a %s. Manage it with the resource for its type.", id, d.Type, r.kind.apiType))
		return false
	}
	m.Name = types.StringValue(d.Name)
	m.Status = types.StringValue(d.Status)
	applyOutputs(m, d.Outputs, r.kind)
	settleUnknowns(m)
	// After an import only the id is known: take the identifiers from the
	// deployment record when it carries them.
	if d.Raw != nil {
		fill := func(dst *types.String, key string) {
			if !dst.IsNull() && !dst.IsUnknown() && dst.ValueString() != "" {
				return
			}
			if v, ok := d.Raw.AdditionalProperties[key].(string); ok && v != "" {
				*dst = types.StringValue(v)
			}
		}
		// git_repo_id is read back; "" means the service has none, which
		// keeps a value recorded here (after an import or a move).
		if ci, ok := d.Raw.AdditionalProperties["ci_config"].(map[string]any); ok && r.kind.has("git_repo_id") {
			if v, ok := ci["git_repo_id"].(string); ok && v != "" {
				m.GitRepoID = types.StringValue(v)
			}
		}
		fill(&m.ServiceID, "service_id")
		// The deployment record names the environment env_id.
		fill(&m.EnvironmentID, "env_id")
		fill(&m.EnvironmentID, "environment_id")
		fill(&m.ProjectID, "project_id")
	}
	if r.kind.has("instance_type") {
		// A configured alias (FLX, MAX, PWR) the API resolved to its code
		// stays as written, so it never shows a diff.
		f := spherelet.ReadDeploymentFields(got.Body)
		m.InstanceType = spherelet.Value(m.InstanceType, f.InstanceType, f.InstanceTypePresent,
			spherelet.AliasLookup(ctx, r.client))
	}
	if st, err := r.client.GetDeploymentStatusWithResponse(ctx, id); err == nil && st.StatusCode() == http.StatusOK && st.JSON200 != nil {
		m.Status = types.StringValue(st.JSON200.Status)
	}
	return true
}

func (r *computeResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	state := getModel(ctx, req.State.Get, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}
	id, err := uuid.Parse(state.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Invalid deployment id", err.Error())
		return
	}
	if !r.refresh(ctx, id, &state, &resp.Diagnostics) {
		if !resp.Diagnostics.HasError() {
			resp.State.RemoveResource(ctx)
		}
		return
	}
	if state.WaitForReady.IsNull() {
		state.WaitForReady = types.BoolValue(true)
	}
	r.setModel(ctx, resp.State.Set, state, &resp.Diagnostics)
}

func (r *computeResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	plan := getModel(ctx, req.Plan.Get, &resp.Diagnostics)
	state := getModel(ctx, req.State.Get, &resp.Diagnostics)
	r.withConfigPassword(ctx, req.Config, &plan, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}
	id, err := uuid.Parse(state.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Invalid deployment id", err.Error())
		return
	}
	plan.ID = state.ID
	// Values unknown at plan time are checked here, before any API call.
	r.refuseGitRepoIDChange(ctx, req.Plan, req.State, &resp.Diagnostics)
	r.refuseGitToImage(ctx, req.Plan, req.State, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}

	if r.kind.has("instance_type") && plan.InstanceType.IsUnknown() {
		// Not configured and not known before: keep what state has; the read
		// after the deploy fills it in.
		plan.InstanceType = state.InstanceType
	}

	// Every write saves its change without rolling it out (skip_redeploy);
	// one deploy at the end rolls out everything this apply changed.
	// A git-built service's build settings take effect only through a build
	// (with skip_redeploy the change body starts none, and /deploy rolls out
	// the image already running), so the apply ends with a build instead.
	// A static site's deploy is a build.
	wrote, rebuild := false, false
	if body := changeBody(ctx, r.kind, plan, state, &resp.Diagnostics); len(body) > 0 {
		if resp.Diagnostics.HasError() || !r.patch(ctx, id, body, &resp.Diagnostics) {
			return
		}
		_, hasBuild := body["build_config"]
		rebuild = hasBuild && !r.kind.is(staticSite) && gitBuilt(r.kind, plan)
		wrote = true
	}
	if !plan.EnvVars.Equal(state.EnvVars) || !plan.SecretVars.Equal(state.SecretVars) {
		if !r.setEnvVars(ctx, id, plan, state, &resp.Diagnostics) {
			return
		}
		wrote = true
	}
	count, size := types.Int64Null(), types.StringNull()
	if r.kind.has("sphere_count") && present(plan.SphereCount) && !plan.SphereCount.Equal(state.SphereCount) {
		count = plan.SphereCount
	}
	if r.kind.has("instance_type") && present(plan.InstanceType) && !plan.InstanceType.Equal(state.InstanceType) {
		size = plan.InstanceType
	}
	if !count.IsNull() || !size.IsNull() {
		if !r.scale(ctx, id, count, size, &resp.Diagnostics) {
			return
		}
		wrote = true
	}
	if wrote {
		rollout := r.deploy
		if rebuild {
			rollout = r.build
		}
		if !rollout(ctx, id, &resp.Diagnostics) {
			return
		}
		if waitFor(plan) && !r.waitForReady(ctx, id, &resp.Diagnostics) {
			return
		}
	}
	r.refresh(ctx, id, &plan, &resp.Diagnostics)
	r.setModel(ctx, resp.State.Set, plan, &resp.Diagnostics)
}

func (r *computeResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	state := getModel(ctx, req.State.Get, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}
	id, err := uuid.Parse(state.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Invalid deployment id", err.Error())
		return
	}
	d, err := r.client.DeleteDeploymentWithResponse(ctx, id)
	if err != nil {
		resp.Diagnostics.AddError("Error deleting "+r.kind.noun, err.Error())
		return
	}
	switch d.StatusCode() {
	case http.StatusOK, http.StatusNoContent, http.StatusAccepted, http.StatusNotFound:
		resp.State.RemoveResource(ctx)
	default:
		resp.Diagnostics.AddError("Error deleting "+r.kind.noun, cstypes.ProblemSummary(d.Body, d.StatusCode()))
	}
}

func (r *computeResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}
