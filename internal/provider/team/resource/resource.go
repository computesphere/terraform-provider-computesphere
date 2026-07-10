package provider

import (
	"context"
	"net/http"
	"time"

	csv2 "github.com/computesphere/computesphere-go"
	cstypes "github.com/computesphere/terraform-provider-computesphere/internal/provider/types"
	"github.com/google/uuid"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/identityschema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// TeamResource manages a ComputeSphere team resource.
//
// Example Usage:
//
//	resource "computesphere_team" "default" {
//	  name        = "example-team"
//	  description = "A sample ComputeSphere team created via Terraform"
//	}
type TeamResource struct {
	client    *csv2.ClientWithResponses
	accountID string
}

var _ resource.Resource = &TeamResource{}
var _ resource.ResourceWithImportState = &TeamResource{}
var _ resource.ResourceWithIdentity = &TeamResource{}

func NewTeamResource() resource.Resource {
	return &TeamResource{}
}

func (r *TeamResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = "computesphere_team"
}

func (r *TeamResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = Schema(ctx)
}

type teamResourceModel struct {
	Name        types.String `tfsdk:"name"`
	AccountID   types.String `tfsdk:"account_id"`
	Description types.String `tfsdk:"description"`
	CreatedAt   types.String `tfsdk:"created_at"`
	UpdatedAt   types.String `tfsdk:"updated_at"`
	ID          types.String `tfsdk:"id"`
}

func (r *TeamResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	data := cstypes.ConfigureResource(req, resp)
	if data != nil {
		r.client = data.V2Client
		r.accountID = data.AccountID
	}
}

func (m *teamResourceModel) apply(t *csv2.Team) {
	m.ID = types.StringValue(t.Id)
	m.Name = types.StringValue(t.Name)
	m.AccountID = types.StringValue(t.AccountId.String())
	m.Description = types.StringPointerValue(t.Description)
	m.CreatedAt = types.StringValue(t.CreatedAt.Format(time.RFC3339))
	m.UpdatedAt = types.StringValue(t.UpdatedAt.Format(time.RFC3339))
}

// resolveAccount returns the account id to use: the per-resource override when
// set, otherwise the provider-level account id.
func (r *TeamResource) resolveAccount(override types.String) (uuid.UUID, error) {
	s := r.accountID
	if !override.IsNull() && override.ValueString() != "" {
		s = override.ValueString()
	}
	return uuid.Parse(s)
}

func (r *TeamResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan teamResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	accountID, err := r.resolveAccount(plan.AccountID)
	if err != nil {
		resp.Diagnostics.AddError("Invalid account_id", err.Error())
		return
	}

	apiResp, err := r.client.CreateTeamWithResponse(ctx, csv2.CreateTeamRequest{
		AccountId:   accountID,
		Name:        plan.Name.ValueString(),
		Description: cstypes.StringPtr(plan.Description.ValueString()),
	})
	if err != nil {
		resp.Diagnostics.AddError("Error creating team", err.Error())
		return
	}
	if apiResp.StatusCode() != http.StatusCreated || apiResp.JSON201 == nil {
		resp.Diagnostics.AddError("Error creating team", cstypes.ProblemSummary(apiResp.Body, apiResp.StatusCode()))
		return
	}

	var state teamResourceModel
	state.apply(apiResp.JSON201)
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *TeamResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state teamResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	tid, err := uuid.Parse(state.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Invalid team id", err.Error())
		return
	}

	apiResp, err := r.client.GetTeamWithResponse(ctx, tid)
	if err != nil {
		resp.Diagnostics.AddError("Error reading team", err.Error())
		return
	}
	if apiResp.StatusCode() == http.StatusNotFound {
		resp.State.RemoveResource(ctx)
		return
	}
	if apiResp.StatusCode() != http.StatusOK || apiResp.JSON200 == nil {
		resp.Diagnostics.AddError("Error reading team", cstypes.ProblemSummary(apiResp.Body, apiResp.StatusCode()))
		return
	}

	state.apply(apiResp.JSON200)
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *TeamResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan teamResourceModel
	var state teamResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	tid, err := uuid.Parse(state.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Invalid team id", err.Error())
		return
	}

	name := plan.Name.ValueString()
	apiResp, err := r.client.UpdateTeamWithResponse(ctx, tid, csv2.UpdateTeamRequest{
		Name:        &name,
		Description: cstypes.StringPtr(plan.Description.ValueString()),
	})
	if err != nil {
		resp.Diagnostics.AddError("Error updating team", err.Error())
		return
	}
	if apiResp.StatusCode() != http.StatusOK || apiResp.JSON200 == nil {
		resp.Diagnostics.AddError("Error updating team", cstypes.ProblemSummary(apiResp.Body, apiResp.StatusCode()))
		return
	}

	var newState teamResourceModel
	newState.apply(apiResp.JSON200)
	resp.Diagnostics.Append(resp.State.Set(ctx, &newState)...)
}

func (r *TeamResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state teamResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	tid, err := uuid.Parse(state.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Invalid team id", err.Error())
		return
	}

	apiResp, err := r.client.DeleteTeamWithResponse(ctx, tid)
	if err != nil {
		resp.Diagnostics.AddError("Error deleting team", err.Error())
		return
	}
	switch apiResp.StatusCode() {
	case http.StatusOK, http.StatusNoContent, http.StatusAccepted, http.StatusNotFound:
		resp.State.RemoveResource(ctx)
	default:
		resp.Diagnostics.AddError("Error deleting team", cstypes.ProblemSummary(apiResp.Body, apiResp.StatusCode()))
	}
}

func (r *TeamResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

func (r *TeamResource) IdentitySchema(ctx context.Context, req resource.IdentitySchemaRequest, resp *resource.IdentitySchemaResponse) {
	resp.IdentitySchema = identityschema.Schema{
		Attributes: map[string]identityschema.Attribute{
			"id": identityschema.StringAttribute{RequiredForImport: true},
		},
	}
}
