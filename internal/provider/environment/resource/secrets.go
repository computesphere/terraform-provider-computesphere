package provider

import (
	"context"
	"net/http"

	csv2 "github.com/computesphere/computesphere-go"
	cstypes "github.com/computesphere/terraform-provider-computesphere/internal/provider/types"
	"github.com/google/uuid"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// EnvironmentSecretsResource manages an environment's shared secrets: the
// baseline every deployment in the environment inherits per key. The API
// never returns secret values, so state keeps the configured values; the
// resource owns the set of names (a secret set elsewhere and not listed here
// is removed).
type EnvironmentSecretsResource struct {
	client *csv2.ClientWithResponses
}

var _ resource.Resource = &EnvironmentSecretsResource{}
var _ resource.ResourceWithImportState = &EnvironmentSecretsResource{}

func NewEnvironmentSecretsResource() resource.Resource {
	return &EnvironmentSecretsResource{}
}

type environmentSecretsResourceModel struct {
	ID            types.String `tfsdk:"id"`
	EnvironmentID types.String `tfsdk:"environment_id"`
	Secrets       types.Map    `tfsdk:"secrets"`
}

func (r *EnvironmentSecretsResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = "computesphere_environment_secrets"
}

func (r *EnvironmentSecretsResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages an environment's shared secrets: the secrets every deployment in the environment " +
			"inherits (a deployment's own value for a key wins). Values are write-only: the API never returns them, " +
			"so a value changed outside Terraform isn't detected. This resource owns the set of names: a secret set " +
			"elsewhere and not listed here is removed. Changing a secret restarts the deployments that inherit it.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed: true, Description: "The environment id.",
				PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"environment_id": schema.StringAttribute{
				Required: true, Description: "Environment the secrets belong to. Changing it removes the secrets from the old environment and sets them on the new one.",
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			"secrets": schema.MapAttribute{
				Required: true, Sensitive: true, ElementType: types.StringType,
				Description: "Secret name to value. An empty map removes every shared secret of the environment.",
			},
		},
	}
}

func (r *EnvironmentSecretsResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if data := cstypes.ConfigureResource(req, resp); data != nil {
		r.client = data.V2Client
	}
}

// keepValues maps the names the API returns to state: a name the prior value
// holds keeps its value (the API returns none), a new name reads as "" so the
// plan shows it, and a name the API no longer has is dropped.
func keepValues(ctx context.Context, names map[string]string, prior types.Map, diags *diag.Diagnostics) types.Map {
	old := map[string]string{}
	if !prior.IsNull() && !prior.IsUnknown() {
		diags.Append(prior.ElementsAs(ctx, &old, false)...)
	}
	out := map[string]string{}
	for k := range names {
		out[k] = old[k]
	}
	v, d := types.MapValueFrom(ctx, types.StringType, out)
	diags.Append(d...)
	return v
}

// write replaces the environment's secrets with the plan's. A secret whose
// value is the same as in prior (the state) is sent blank, which keeps the
// stored value, so unchanged values aren't sent again.
func (r *EnvironmentSecretsResource) write(ctx context.Context, plan environmentSecretsResourceModel, prior types.Map, diags *diag.Diagnostics) *environmentSecretsResourceModel {
	eid, err := uuid.Parse(plan.EnvironmentID.ValueString())
	if err != nil {
		diags.AddAttributeError(path.Root("environment_id"), "Invalid environment id", err.Error())
		return nil
	}
	secrets := map[string]string{}
	diags.Append(plan.Secrets.ElementsAs(ctx, &secrets, false)...)
	if diags.HasError() {
		return nil
	}
	old := map[string]string{}
	if !prior.IsNull() && !prior.IsUnknown() {
		diags.Append(prior.ElementsAs(ctx, &old, false)...)
	}
	send := map[string]string{}
	for k, v := range secrets {
		if ov, ok := old[k]; ok && ov == v && v != "" {
			v = ""
		}
		send[k] = v
	}
	var editors []csv2.RequestEditorFn
	if len(secrets) == 0 {
		editors = append(editors, withBodyField("allow_empty", true))
	}
	apiResp, err := r.client.ReplaceEnvironmentSecretsWithResponse(ctx, eid, csv2.SetEnvironmentSecretsRequest{Secrets: send}, editors...)
	if err != nil {
		diags.AddError("Error setting environment secrets", err.Error())
		return nil
	}
	if apiResp.StatusCode() != http.StatusOK || apiResp.JSON200 == nil {
		diags.AddError("Error setting environment secrets", cstypes.ProblemSummary(apiResp.Body, apiResp.StatusCode()))
		return nil
	}
	state := environmentSecretsResourceModel{
		ID: types.StringValue(eid.String()), EnvironmentID: types.StringValue(eid.String()),
		Secrets: keepValues(ctx, apiResp.JSON200.Secrets, plan.Secrets, diags),
	}
	return &state
}

func (r *EnvironmentSecretsResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan environmentSecretsResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	if state := r.write(ctx, plan, types.MapNull(types.StringType), &resp.Diagnostics); state != nil {
		resp.Diagnostics.Append(resp.State.Set(ctx, state)...)
	}
}

func (r *EnvironmentSecretsResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan, prior environmentSecretsResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &prior)...)
	if resp.Diagnostics.HasError() {
		return
	}
	if state := r.write(ctx, plan, prior.Secrets, &resp.Diagnostics); state != nil {
		resp.Diagnostics.Append(resp.State.Set(ctx, state)...)
	}
}

func (r *EnvironmentSecretsResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state environmentSecretsResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	eid, err := uuid.Parse(state.EnvironmentID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Invalid environment id", err.Error())
		return
	}
	apiResp, err := r.client.GetEnvironmentSecretsWithResponse(ctx, eid)
	if err != nil {
		resp.Diagnostics.AddError("Error reading environment secrets", err.Error())
		return
	}
	if apiResp.StatusCode() == http.StatusNotFound {
		resp.State.RemoveResource(ctx)
		return
	}
	if apiResp.StatusCode() != http.StatusOK || apiResp.JSON200 == nil {
		resp.Diagnostics.AddError("Error reading environment secrets", cstypes.ProblemSummary(apiResp.Body, apiResp.StatusCode()))
		return
	}
	state.ID = types.StringValue(eid.String())
	state.Secrets = keepValues(ctx, apiResp.JSON200.Secrets, state.Secrets, &resp.Diagnostics)
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

// Delete removes every shared secret of the environment.
func (r *EnvironmentSecretsResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state environmentSecretsResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	eid, err := uuid.Parse(state.EnvironmentID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Invalid environment id", err.Error())
		return
	}
	apiResp, err := r.client.ReplaceEnvironmentSecretsWithResponse(ctx, eid,
		csv2.SetEnvironmentSecretsRequest{Secrets: map[string]string{}}, withBodyField("allow_empty", true))
	if err != nil {
		resp.Diagnostics.AddError("Error removing environment secrets", err.Error())
		return
	}
	switch apiResp.StatusCode() {
	case http.StatusOK, http.StatusNotFound:
	default:
		resp.Diagnostics.AddError("Error removing environment secrets", cstypes.ProblemSummary(apiResp.Body, apiResp.StatusCode()))
	}
}

// ImportState imports by environment id. Values aren't returned by the API,
// so the first plan after an import sets every secret again.
func (r *EnvironmentSecretsResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("environment_id"), req, resp)
}
