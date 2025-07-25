package provider

import (
	"context"

	cs "github.com/computesphere/cli/cs"
	cstypes "github.com/computesphere/terraform-provider-computesphere/internal/provider/types"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/identityschema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

type EnvironmentResource struct {
	client *cs.APIClient
}

var _ resource.Resource = &EnvironmentResource{}
var _ resource.ResourceWithIdentity = &EnvironmentResource{}

func NewEnvironmentResource() resource.Resource {
	return &EnvironmentResource{}
}

func (r *EnvironmentResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = "computesphere_environment"
}

func (r *EnvironmentResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = Schema(ctx)
}

func (r *EnvironmentResource) IdentitySchema(ctx context.Context, req resource.IdentitySchemaRequest, resp *resource.IdentitySchemaResponse) {
	resp.IdentitySchema = identityschema.Schema{
		Attributes: map[string]identityschema.Attribute{
			"id": identityschema.StringAttribute{RequiredForImport: true},
		},
	}
}

type environmentResourceModel struct {
	Name      types.String `tfsdk:"name"`
	Region    types.String `tfsdk:"region"`
	ProjectID types.String `tfsdk:"project_id"`
	ID        types.String `tfsdk:"id"`
	CreatedAt types.String `tfsdk:"created_at"`
}

func (r *EnvironmentResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	data := cstypes.ConfigureResource(req, resp)
	if data != nil {
		r.client = data.Client
	}
}

func (r *EnvironmentResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan environmentResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	payload := cs.ModelEnvironmentRequest{
		Name:      &[]string{plan.Name.ValueString()}[0],
		Region:    &[]string{plan.Region.ValueString()}[0],
		ProjectId: &[]string{plan.ProjectID.ValueString()}[0],
	}
	apiResp, _, err := r.client.EnvironmentAPI.EnvironmentsPost(ctx).Body(payload).Execute()
	if err != nil {
		resp.Diagnostics.AddError("Error creating environment", err.Error())
		return
	}
	if apiResp.Data == nil || apiResp.Data.Id == nil {
		resp.Diagnostics.AddError("Environment creation failed", "No ID returned")
		return
	}
	plan.ID = types.StringValue(*apiResp.Data.Id)
	plan.CreatedAt = types.StringValue(apiResp.Data.GetCreatedAt())
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *EnvironmentResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state environmentResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	apiResp, httpResp, err := r.client.EnvironmentAPI.EnvironmentsIdGet(ctx, state.ID.ValueString()).Execute()
	if err != nil {
		if httpResp != nil && httpResp.StatusCode == 404 {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Error reading environment", err.Error())
		return
	}
	if apiResp.Data == nil {
		resp.State.RemoveResource(ctx)
		return
	}
	env := apiResp.Data
	state.Name = types.StringValue(env.GetName())
	state.Region = types.StringValue(env.GetRegion())
	state.ProjectID = types.StringValue(env.GetProjectId())
	state.CreatedAt = types.StringValue(env.GetCreatedAt())
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *EnvironmentResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan environmentResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	payload := cs.ModelEnvironmentUpdateRequest{
		Name: &[]string{plan.Name.ValueString()}[0],
	}
	_, _, err := r.client.EnvironmentAPI.EnvironmentsIdPut(ctx, plan.ID.ValueString()).Body(payload).Execute()
	if err != nil {
		resp.Diagnostics.AddError("Error updating environment", err.Error())
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *EnvironmentResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state environmentResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	_, httpResp, err := r.client.EnvironmentAPI.EnvironmentsIdDelete(ctx, state.ID.ValueString()).Execute()
	if err != nil {
		if httpResp != nil && httpResp.StatusCode == 404 {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Error deleting environment", err.Error())
		return
	}
	resp.State.RemoveResource(ctx)
}
