package provider

import (
	"context"
	"net/http"
	"time"

	csv2 "github.com/computesphere/computesphere-go"
	cstypes "github.com/computesphere/terraform-provider-computesphere/internal/provider/types"
	"github.com/google/uuid"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/identityschema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

type ServiceResource struct {
	client    *csv2.ClientWithResponses
	accountID string
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

// serviceResourceModel — env_id/image_name/port/sphere_count are intentionally
// absent: under the compute model those are deployment-level concerns, not part
// of the v2 service resource. A service carries its plan (spherelet shape).
type serviceResourceModel struct {
	Name         types.String `tfsdk:"name"`
	ProjectID    types.String `tfsdk:"project_id"`
	Type         types.String `tfsdk:"type"`
	PlanID       types.String `tfsdk:"plan_id"`
	Active       types.Bool   `tfsdk:"active"`
	CreatedAt    types.String `tfsdk:"created_at"`
	LastOpenedAt types.String `tfsdk:"last_opened_at"`
	ID           types.String `tfsdk:"id"`
}

func (r *ServiceResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	data := cstypes.ConfigureResource(req, resp)
	if data != nil {
		r.client = data.V2Client
		r.accountID = data.AccountID
	}
}

func (m *serviceResourceModel) apply(s *csv2.Service) {
	m.ID = types.StringValue(s.Id)
	m.Name = types.StringValue(s.Name)
	m.ProjectID = types.StringValue(s.ProjectId.String())
	m.Type = types.StringValue(s.Type)
	m.PlanID = types.StringPointerValue(s.PlanId)
	m.Active = types.BoolValue(s.Active)
	m.CreatedAt = types.StringValue(s.CreatedAt.Format(time.RFC3339))
	m.LastOpenedAt = cstypes.TimePtrString(s.LastOpenedAt)
}

func (r *ServiceResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan serviceResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	projectID, err := uuid.Parse(plan.ProjectID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Invalid project_id", err.Error())
		return
	}

	apiResp, err := r.client.CreateServiceWithResponse(ctx, csv2.CreateServiceRequest{
		Name:      plan.Name.ValueString(),
		ProjectId: projectID,
		Type:      plan.Type.ValueString(),
		PlanId:    cstypes.StringPtr(plan.PlanID.ValueString()),
	})
	if err != nil {
		resp.Diagnostics.AddError("Error creating service", err.Error())
		return
	}
	if apiResp.StatusCode() != http.StatusCreated || apiResp.JSON201 == nil {
		resp.Diagnostics.AddError("Error creating service", cstypes.ProblemSummary(apiResp.Body, apiResp.StatusCode()))
		return
	}

	var state serviceResourceModel
	state.apply(apiResp.JSON201)
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *ServiceResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state serviceResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	sid, err := uuid.Parse(state.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Invalid service id", err.Error())
		return
	}

	apiResp, err := r.client.GetServiceWithResponse(ctx, csv2.ServiceId(sid))
	if err != nil {
		resp.Diagnostics.AddError("Error reading service", err.Error())
		return
	}
	if apiResp.StatusCode() == http.StatusNotFound {
		resp.State.RemoveResource(ctx)
		return
	}
	if apiResp.StatusCode() != http.StatusOK || apiResp.JSON200 == nil {
		resp.Diagnostics.AddError("Error reading service", cstypes.ProblemSummary(apiResp.Body, apiResp.StatusCode()))
		return
	}

	state.apply(apiResp.JSON200)
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *ServiceResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan serviceResourceModel
	var state serviceResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	sid, err := uuid.Parse(state.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Invalid service id", err.Error())
		return
	}

	// The v2 service update surface is the plan (spherelet shape); name/type/
	// project changes force replacement via RequiresReplace.
	apiResp, err := r.client.UpdateServiceWithResponse(ctx, csv2.ServiceId(sid), csv2.UpdateServiceRequest{
		PlanId: plan.PlanID.ValueString(),
	})
	if err != nil {
		resp.Diagnostics.AddError("Error updating service", err.Error())
		return
	}
	if apiResp.StatusCode() != http.StatusOK || apiResp.JSON200 == nil {
		resp.Diagnostics.AddError("Error updating service", cstypes.ProblemSummary(apiResp.Body, apiResp.StatusCode()))
		return
	}

	var newState serviceResourceModel
	newState.apply(apiResp.JSON200)
	resp.Diagnostics.Append(resp.State.Set(ctx, &newState)...)
}

func (r *ServiceResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state serviceResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	sid, err := uuid.Parse(state.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Invalid service id", err.Error())
		return
	}

	apiResp, err := r.client.DeleteServiceWithResponse(ctx, csv2.ServiceId(sid))
	if err != nil {
		resp.Diagnostics.AddError("Error deleting service", err.Error())
		return
	}
	switch apiResp.StatusCode() {
	case http.StatusOK, http.StatusNoContent, http.StatusAccepted, http.StatusNotFound:
		resp.State.RemoveResource(ctx)
	default:
		resp.Diagnostics.AddError("Error deleting service", cstypes.ProblemSummary(apiResp.Body, apiResp.StatusCode()))
	}
}
