package provider

import (
	"context"

	cs "github.com/computesphere/cli/cs"
	cstypes "github.com/computesphere/terraform-provider-computesphere/internal/provider/types"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/identityschema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

type ServiceResource struct {
	client *cs.APIClient
}

var _ resource.Resource = &ServiceResource{}
var _ resource.ResourceWithIdentity = &ServiceResource{}

func NewServiceResource() resource.Resource {
	return &ServiceResource{}
}

func (r *ServiceResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = "computesphere_service"
}

func (r *ServiceResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = Schema(ctx)
}

func (r *ServiceResource) IdentitySchema(ctx context.Context, req resource.IdentitySchemaRequest, resp *resource.IdentitySchemaResponse) {
	resp.IdentitySchema = identityschema.Schema{
		Attributes: map[string]identityschema.Attribute{
			"id": identityschema.StringAttribute{RequiredForImport: true},
		},
	}
}

type serviceResourceModel struct {
	Name        types.String `tfsdk:"name"`
	ProjectID   types.String `tfsdk:"project_id"`
	EnvID       types.String `tfsdk:"env_id"`
	ImageName   types.String `tfsdk:"image_name"`
	Port        types.Int64  `tfsdk:"port"`
	SphereCount types.Int64  `tfsdk:"sphere_count"`
	Type        types.String `tfsdk:"type"`
	ID          types.String `tfsdk:"id"`
	CreatedAt   types.String `tfsdk:"created_at"`
}

func (r *ServiceResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	data := cstypes.ConfigureResource(req, resp)
	if data != nil {
		r.client = data.Client
	}
}

func (r *ServiceResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan serviceResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	payload := cs.ModelServiceRequest{
		Name:      &[]string{plan.Name.ValueString()}[0],
		ProjectId: &[]string{plan.ProjectID.ValueString()}[0],
	}
	if !plan.EnvID.IsNull() {
		envID := plan.EnvID.ValueString()
		payload.EnvId = &envID
	}
	if !plan.ImageName.IsNull() {
		imageName := plan.ImageName.ValueString()
		payload.ImageName = &imageName
	}
	if !plan.Port.IsNull() {
		port := int(plan.Port.ValueInt64())
		payload.Port = &port
	}
	if !plan.SphereCount.IsNull() {
		count := int(plan.SphereCount.ValueInt64())
		payload.SphereCount = &count
	}
	if !plan.Type.IsNull() {
		typeVal := plan.Type.ValueString()
		payload.Type = &typeVal
	}
	apiResp, _, err := r.client.ServiceAPI.ServicesPost(ctx).Body(payload).Execute()
	if err != nil {
		resp.Diagnostics.AddError("Error creating service", err.Error())
		return
	}
	if apiResp.Data == nil || apiResp.Data.Id == nil {
		resp.Diagnostics.AddError("Service creation failed", "No ID returned")
		return
	}
	plan.ID = types.StringValue(*apiResp.Data.Id)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
	r.Read(ctx, resource.ReadRequest{State: resp.State}, &resource.ReadResponse{State: resp.State, Diagnostics: resp.Diagnostics})
}

func (r *ServiceResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state serviceResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	apiResp, httpResp, err := r.client.ServiceAPI.ServicesIdGet(ctx, state.ID.ValueString()).Execute()
	if err != nil {
		if httpResp != nil && httpResp.StatusCode == 404 {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Error reading service", err.Error())
		return
	}
	if apiResp.Data == nil {
		resp.State.RemoveResource(ctx)
		return
	}
	service := apiResp.Data
	state.Name = types.StringValue(service.GetName())
	state.ProjectID = types.StringValue(service.GetProjectId())
	state.CreatedAt = types.StringValue(service.GetCreatedAt())
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *ServiceResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan serviceResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	payload := cs.ModelServiceReDeployRequest{}
	if !plan.EnvID.IsNull() {
		envID := plan.EnvID.ValueString()
		payload.EnvId = &envID
	}
	if !plan.ImageName.IsNull() {
		imageName := plan.ImageName.ValueString()
		payload.ImageName = &imageName
	}
	if !plan.Port.IsNull() {
		port := int(plan.Port.ValueInt64())
		payload.Port = &port
	}
	if !plan.SphereCount.IsNull() {
		count := int(plan.SphereCount.ValueInt64())
		payload.SphereCount = &count
	}
	_, _, err := r.client.ServiceAPI.ServicesIdPost(ctx, plan.ID.ValueString()).Body(payload).Execute()
	if err != nil {
		resp.Diagnostics.AddError("Error updating service", err.Error())
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
	r.Read(ctx, resource.ReadRequest{State: resp.State}, &resource.ReadResponse{State: resp.State, Diagnostics: resp.Diagnostics})
}

func (r *ServiceResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state serviceResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	_, httpResp, err := r.client.ServiceAPI.ServicesIdDelete(ctx, state.ID.ValueString()).Execute()
	if err != nil {
		if httpResp != nil && httpResp.StatusCode == 404 {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Error deleting service", err.Error())
		return
	}
	resp.State.RemoveResource(ctx)
}
