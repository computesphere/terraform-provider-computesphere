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

type EnvironmentResource struct {
	client    *csv2.ClientWithResponses
	accountID string
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
		r.client = data.V2Client
		r.accountID = data.AccountID
	}
}

func (m *environmentResourceModel) apply(e *csv2.Environment) {
	m.ID = types.StringValue(e.Id)
	m.Name = types.StringValue(e.Name)
	m.Region = types.StringValue(e.Region)
	m.ProjectID = types.StringValue(e.ProjectId.String())
	m.CreatedAt = types.StringValue(e.CreatedAt.Format(time.RFC3339))
}

func (r *EnvironmentResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan environmentResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	projectID, err := uuid.Parse(plan.ProjectID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Invalid project_id", err.Error())
		return
	}

	apiResp, err := r.client.CreateEnvironmentWithResponse(ctx, csv2.CreateEnvironmentRequest{
		Name:      plan.Name.ValueString(),
		ProjectId: projectID,
		Region:    plan.Region.ValueString(),
	})
	if err != nil {
		resp.Diagnostics.AddError("Error creating environment", err.Error())
		return
	}
	if apiResp.StatusCode() != http.StatusCreated || apiResp.JSON201 == nil {
		resp.Diagnostics.AddError("Error creating environment", cstypes.ProblemSummary(apiResp.Body, apiResp.StatusCode()))
		return
	}

	var state environmentResourceModel
	state.apply(apiResp.JSON201)
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *EnvironmentResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state environmentResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	eid, err := uuid.Parse(state.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Invalid environment id", err.Error())
		return
	}

	apiResp, err := r.client.GetEnvironmentWithResponse(ctx, eid)
	if err != nil {
		resp.Diagnostics.AddError("Error reading environment", err.Error())
		return
	}
	if apiResp.StatusCode() == http.StatusNotFound {
		resp.State.RemoveResource(ctx)
		return
	}
	if apiResp.StatusCode() != http.StatusOK || apiResp.JSON200 == nil {
		resp.Diagnostics.AddError("Error reading environment", cstypes.ProblemSummary(apiResp.Body, apiResp.StatusCode()))
		return
	}

	state.apply(apiResp.JSON200)
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *EnvironmentResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan environmentResourceModel
	var state environmentResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	eid, err := uuid.Parse(state.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Invalid environment id", err.Error())
		return
	}

	name := plan.Name.ValueString()
	apiResp, err := r.client.UpdateEnvironmentWithResponse(ctx, eid, csv2.UpdateEnvironmentRequest{
		Name: &name,
	})
	if err != nil {
		resp.Diagnostics.AddError("Error updating environment", err.Error())
		return
	}
	if apiResp.StatusCode() != http.StatusOK || apiResp.JSON200 == nil {
		resp.Diagnostics.AddError("Error updating environment", cstypes.ProblemSummary(apiResp.Body, apiResp.StatusCode()))
		return
	}

	var newState environmentResourceModel
	newState.apply(apiResp.JSON200)
	resp.Diagnostics.Append(resp.State.Set(ctx, &newState)...)
}

func (r *EnvironmentResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state environmentResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	eid, err := uuid.Parse(state.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Invalid environment id", err.Error())
		return
	}

	cascade := true
	apiResp, err := r.client.DeleteEnvironmentWithResponse(ctx, eid, &csv2.DeleteEnvironmentParams{
		Cascade: &cascade,
	})
	if err != nil {
		resp.Diagnostics.AddError("Error deleting environment", err.Error())
		return
	}
	switch apiResp.StatusCode() {
	case http.StatusOK, http.StatusNoContent, http.StatusAccepted, http.StatusNotFound:
		resp.State.RemoveResource(ctx)
	default:
		resp.Diagnostics.AddError("Error deleting environment", cstypes.ProblemSummary(apiResp.Body, apiResp.StatusCode()))
	}
}
