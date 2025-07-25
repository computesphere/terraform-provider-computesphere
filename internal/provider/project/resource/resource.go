package provider

import (
	"context"

	cs "github.com/computesphere/cli/cs"
	cstypes "github.com/computesphere/terraform-provider-computesphere/internal/provider/types"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/identityschema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

type ProjectResource struct {
	client *cs.APIClient
}

var _ resource.Resource = &ProjectResource{}
var _ resource.ResourceWithImportState = &ProjectResource{}
var _ resource.ResourceWithIdentity = &ProjectResource{}

func NewProjectResource() resource.Resource {
	return &ProjectResource{}
}

func (r *ProjectResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = "computesphere_project"
}

func (r *ProjectResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = Schema(ctx)
}

type projectResourceModel struct {
	Name        types.String `tfsdk:"name"`
	Description types.String `tfsdk:"description"`
	PlanName    types.String `tfsdk:"plan_name"`
	PlanValue   types.Int64  `tfsdk:"plan_value"`
	PlanID      types.String `tfsdk:"plan_id"`
	CreatedAt   types.String `tfsdk:"created_at"`
	ID          types.String `tfsdk:"id"`
}

func (r *ProjectResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	data := cstypes.ConfigureResource(req, resp)
	if data != nil {
		r.client = data.Client
	}
}

func (r *ProjectResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan projectResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...) // get planned values
	if resp.Diagnostics.HasError() {
		return
	}

	// Prepare payload for API
	planName := plan.PlanName.ValueString()
	if planName == "" {
		planName = "FLX"
	}
	payload := cs.ModelProjectRequestPayload{
		Name:        &[]string{plan.Name.ValueString()}[0],
		Description: &[]string{plan.Description.ValueString()}[0],
		PlanName:    &planName,
		PlanValue:   func(v int64) *int { i := int(v); return &i }(plan.PlanValue.ValueInt64()),
	}

	apiResp, _, err := r.client.ProjectAPI.ProjectsPost(ctx).Body(payload).Execute()
	if err != nil {
		resp.Diagnostics.AddError("Error creating project", err.Error())
		return
	}
	if apiResp.Data == nil || apiResp.Data.Id == nil {
		resp.Diagnostics.AddError("Project creation failed", "No ID returned")
		return
	}

	// Map API response to state
	var state projectResourceModel
	state.ID = types.StringValue(*apiResp.Data.Id)
	state.Name = types.StringValue(apiResp.Data.GetName())
	state.Description = types.StringValue(apiResp.Data.GetDescription())
	// Set created_at
	if apiResp.Data.GetCreatedAt() != "" {
		state.CreatedAt = types.StringValue(apiResp.Data.GetCreatedAt())
	} else {
		state.CreatedAt = types.StringValue("")
	}
	// Set plan_id
	if apiResp.Data.Plan != nil && apiResp.Data.Plan.Id != nil {
		state.PlanID = types.StringPointerValue(apiResp.Data.Plan.Id)
	} else if apiResp.Data.PlanId != nil {
		state.PlanID = types.StringPointerValue(apiResp.Data.PlanId)
	} else {
		state.PlanID = types.StringValue("")
	}
	if apiResp.Data.Plan != nil {
		state.PlanName = types.StringPointerValue(apiResp.Data.Plan.Name)
		state.PlanValue = types.Int64Value(int64(apiResp.Data.GetPlanValue()))
	} else {
		state.PlanName = types.StringNull()
		state.PlanValue = types.Int64Null()
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *ProjectResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state projectResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...) // get current state
	if resp.Diagnostics.HasError() {
		return
	}
	apiResp, httpResp, err := r.client.ProjectAPI.ProjectsIdGet(ctx, state.ID.ValueString()).Execute()
	if err != nil {
		if httpResp != nil && httpResp.StatusCode == 404 {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Error reading project", err.Error())
		return
	}
	if apiResp.Data == nil {
		resp.State.RemoveResource(ctx)
		return
	}
	project := apiResp.Data
	if project == nil {
		resp.State.RemoveResource(ctx)
		return
	}
	// Defensive: always set all required/computed fields to non-null values
	if project.GetName() != "" {
		state.Name = types.StringValue(project.GetName())
	} else {
		state.Name = types.StringValue("")
	}
	if project.GetId() != "" {
		state.ID = types.StringValue(project.GetId())
	} else {
		state.ID = types.StringValue("")
	}
	if project.GetDescription() != "" {
		state.Description = types.StringValue(project.GetDescription())
	} else {
		state.Description = types.StringValue("")
	}
	if project.GetCreatedAt() != "" {
		state.CreatedAt = types.StringValue(project.GetCreatedAt())
	} else {
		state.CreatedAt = types.StringValue("")
	}
	// Plan fields: check all pointers
	if project.Plan != nil {
		if project.Plan.Id != nil {
			state.PlanID = types.StringPointerValue(project.Plan.Id)
		} else {
			state.PlanID = types.StringValue("")
		}
		if project.Plan.Name != nil {
			state.PlanName = types.StringPointerValue(project.Plan.Name)
		} else {
			state.PlanName = types.StringValue("")
		}
		state.PlanValue = types.Int64Value(int64(project.GetPlanValue()))
	} else {
		state.PlanID = types.StringValue("")
		state.PlanName = types.StringValue("")
		state.PlanValue = types.Int64Value(0)
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...) // update state
}

func (r *ProjectResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan projectResourceModel
	var state projectResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)   // get planned changes
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...) // get current state
	if resp.Diagnostics.HasError() {
		return
	}
	if state.ID.IsNull() || state.ID.ValueString() == "" {
		resp.Diagnostics.AddError("Missing project ID", "Cannot update project because the ID is missing from state.")
		return
	}
	payload := cs.ModelProjectUpdatePayload{
		Name:        &[]string{plan.Name.ValueString()}[0],
		Description: &[]string{plan.Description.ValueString()}[0],
		PlanValue:   func(v int64) *int { i := int(v); return &i }(plan.PlanValue.ValueInt64()),
	}
	_, _, err := r.client.ProjectAPI.ProjectsIdPut(ctx, state.ID.ValueString()).Body(payload).Execute()
	if err != nil {
		resp.Diagnostics.AddError("Error updating project", err.Error())
		return
	}
	// Standard pattern: call Read once to refresh state
	r.Read(ctx, resource.ReadRequest{State: resp.State}, &resource.ReadResponse{State: resp.State, Diagnostics: resp.Diagnostics})
}

func (r *ProjectResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state projectResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	_, httpResp, err := r.client.ProjectAPI.ProjectsIdDelete(ctx, state.ID.ValueString()).Execute()
	if err != nil {
		if httpResp != nil && httpResp.StatusCode == 404 {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Error deleting project", err.Error())
		return
	}
	resp.State.RemoveResource(ctx)
}

func (r *ProjectResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

func (r *ProjectResource) IdentitySchema(ctx context.Context, req resource.IdentitySchemaRequest, resp *resource.IdentitySchemaResponse) {
	resp.IdentitySchema = identityschema.Schema{
		Attributes: map[string]identityschema.Attribute{
			"id": identityschema.StringAttribute{RequiredForImport: true},
		},
	}
}
