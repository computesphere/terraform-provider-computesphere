package provider

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
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

// EnvironmentVariablesResource manages an environment's shared plain
// variables: the baseline every deployment in the environment inherits per
// key (a deployment's own value for a key wins). The resource owns the whole
// map: keys set in the console and not in the configuration are removed.
type EnvironmentVariablesResource struct {
	client *csv2.ClientWithResponses
}

var _ resource.Resource = &EnvironmentVariablesResource{}
var _ resource.ResourceWithImportState = &EnvironmentVariablesResource{}

func NewEnvironmentVariablesResource() resource.Resource {
	return &EnvironmentVariablesResource{}
}

type environmentVariablesResourceModel struct {
	ID            types.String `tfsdk:"id"`
	EnvironmentID types.String `tfsdk:"environment_id"`
	Variables     types.Map    `tfsdk:"variables"`
}

func (r *EnvironmentVariablesResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = "computesphere_environment_variables"
}

func (r *EnvironmentVariablesResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages an environment's shared variables, one resource per environment: the plain " +
			"variables every deployment in the environment inherits (a deployment's own value for a key wins). " +
			"This resource owns the whole set: a variable set elsewhere and not listed here is removed. Changing " +
			"a variable restarts the deployments that inherit it; a service whose deploy policy is `approval` " +
			"gets the change with its next deploy unless a project admin makes it. Keep credentials out of " +
			"plain variables.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed: true, Description: "The environment id.",
				PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"environment_id": schema.StringAttribute{
				Required: true, Description: "Environment the variables belong to. Changing it removes the variables from the old environment and sets them on the new one.",
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			"variables": schema.MapAttribute{
				Required: true, ElementType: types.StringType,
				Description: "Variable name to value. An empty map removes every shared variable of the environment.",
			},
		},
	}
}

func (r *EnvironmentVariablesResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if data := cstypes.ConfigureResource(req, resp); data != nil {
		r.client = data.V2Client
	}
}

// put replaces the environment's variables with vars. An empty map needs
// allow_empty, which the SDK's request type doesn't carry.
func (r *EnvironmentVariablesResource) put(ctx context.Context, eid uuid.UUID, vars map[string]string, diags *diag.Diagnostics) map[string]string {
	if vars == nil {
		vars = map[string]string{}
	}
	var editors []csv2.RequestEditorFn
	if len(vars) == 0 {
		editors = append(editors, withBodyField("allow_empty", true))
	}
	apiResp, err := r.client.ReplaceEnvironmentVariablesWithResponse(ctx, eid, csv2.SetEnvironmentVariablesRequest{Variables: vars}, editors...)
	if err != nil {
		diags.AddError("Error setting environment variables", err.Error())
		return nil
	}
	if apiResp.StatusCode() != http.StatusOK || apiResp.JSON200 == nil {
		diags.AddError("Error setting environment variables", cstypes.ProblemSummary(apiResp.Body, apiResp.StatusCode()))
		return nil
	}
	return apiResp.JSON200.Variables
}

func (m *environmentVariablesResourceModel) set(ctx context.Context, eid string, vars map[string]string, diags *diag.Diagnostics) {
	if vars == nil {
		vars = map[string]string{}
	}
	m.ID = types.StringValue(eid)
	m.EnvironmentID = types.StringValue(eid)
	v, d := types.MapValueFrom(ctx, types.StringType, vars)
	diags.Append(d...)
	m.Variables = v
}

func (r *EnvironmentVariablesResource) write(ctx context.Context, plan environmentVariablesResourceModel, diags *diag.Diagnostics) *environmentVariablesResourceModel {
	eid, err := uuid.Parse(plan.EnvironmentID.ValueString())
	if err != nil {
		diags.AddAttributeError(path.Root("environment_id"), "Invalid environment id", err.Error())
		return nil
	}
	vars := map[string]string{}
	diags.Append(plan.Variables.ElementsAs(ctx, &vars, false)...)
	if diags.HasError() {
		return nil
	}
	got := r.put(ctx, eid, vars, diags)
	if diags.HasError() {
		return nil
	}
	var state environmentVariablesResourceModel
	state.set(ctx, eid.String(), got, diags)
	return &state
}

func (r *EnvironmentVariablesResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan environmentVariablesResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	if state := r.write(ctx, plan, &resp.Diagnostics); state != nil {
		resp.Diagnostics.Append(resp.State.Set(ctx, state)...)
	}
}

func (r *EnvironmentVariablesResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan environmentVariablesResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	if state := r.write(ctx, plan, &resp.Diagnostics); state != nil {
		resp.Diagnostics.Append(resp.State.Set(ctx, state)...)
	}
}

func (r *EnvironmentVariablesResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state environmentVariablesResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	eid, err := uuid.Parse(state.EnvironmentID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Invalid environment id", err.Error())
		return
	}
	apiResp, err := r.client.GetEnvironmentVariablesWithResponse(ctx, eid)
	if err != nil {
		resp.Diagnostics.AddError("Error reading environment variables", err.Error())
		return
	}
	if apiResp.StatusCode() == http.StatusNotFound {
		resp.State.RemoveResource(ctx)
		return
	}
	if apiResp.StatusCode() != http.StatusOK || apiResp.JSON200 == nil {
		resp.Diagnostics.AddError("Error reading environment variables", cstypes.ProblemSummary(apiResp.Body, apiResp.StatusCode()))
		return
	}
	state.set(ctx, eid.String(), apiResp.JSON200.Variables, &resp.Diagnostics)
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

// Delete removes every shared variable of the environment.
func (r *EnvironmentVariablesResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state environmentVariablesResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	eid, err := uuid.Parse(state.EnvironmentID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Invalid environment id", err.Error())
		return
	}
	apiResp, err := r.client.ReplaceEnvironmentVariablesWithResponse(ctx, eid,
		csv2.SetEnvironmentVariablesRequest{Variables: map[string]string{}}, withBodyField("allow_empty", true))
	if err != nil {
		resp.Diagnostics.AddError("Error removing environment variables", err.Error())
		return
	}
	switch apiResp.StatusCode() {
	case http.StatusOK, http.StatusNotFound:
	default:
		resp.Diagnostics.AddError("Error removing environment variables", cstypes.ProblemSummary(apiResp.Body, apiResp.StatusCode()))
	}
}

// ImportState imports by environment id.
func (r *EnvironmentVariablesResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("environment_id"), req, resp)
}

// withBodyField sets one top-level field in a JSON request body, for fields
// the SDK's request types don't carry.
func withBodyField(name string, value any) csv2.RequestEditorFn {
	return func(_ context.Context, req *http.Request) error {
		if req.Body == nil {
			return nil
		}
		raw, err := io.ReadAll(req.Body)
		if err != nil {
			return err
		}
		body := map[string]any{}
		if len(bytes.TrimSpace(raw)) > 0 {
			if err := json.Unmarshal(raw, &body); err != nil {
				return err
			}
		}
		body[name] = value
		out, err := json.Marshal(body)
		if err != nil {
			return err
		}
		req.Body = io.NopCloser(bytes.NewReader(out))
		req.ContentLength = int64(len(out))
		req.GetBody = func() (io.ReadCloser, error) { return io.NopCloser(bytes.NewReader(out)), nil }
		return nil
	}
}
