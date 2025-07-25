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

// TeamResource manages a ComputeSphere team resource.
//
// Example Usage:
//
//	resource "computesphere_team" "default" {
//	  name        = "example-team"
//	  description = "A sample ComputeSphere team created via Terraform"
//	}
//
//	resource "computesphere_team" "with_account" {
//	  name        = "cross-account-team"
//	  account_id  = "your-account-id"
//	  description = "A team in a specific account"
//	}
type TeamResource struct {
	client *cs.APIClient
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
	ID          types.String `tfsdk:"id"`
}

func (r *TeamResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	data := cstypes.ConfigureResource(req, resp)
	if data != nil {
		r.client = data.Client
	}
}

func (r *TeamResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan teamResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	payload := cs.ModelTeamRequestPayload{
		Name:        &[]string{plan.Name.ValueString()}[0],
		Description: &[]string{plan.Description.ValueString()}[0],
	}
	// Use resource account_id if set, else use provider's account_id from client config
	accountID := plan.AccountID.ValueString()
	if plan.AccountID.IsNull() || accountID == "" {
		// fallback to provider's account_id from client config
		accountID = r.client.GetConfig().DefaultHeader["X-Account-ID"]
	}
	apiResp, _, err := r.client.TeamAPI.TeamsPost(ctx).Body(payload).XAccountId(accountID).Execute()
	if err != nil {
		resp.Diagnostics.AddError("Error creating team", err.Error())
		return
	}
	if apiResp.Data == nil {
		resp.Diagnostics.AddError("Team creation failed", "No ID returned")
		return
	}
	plan.ID = types.StringValue(*apiResp.Data) // team ID only
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
	r.Read(ctx, resource.ReadRequest{State: resp.State}, &resource.ReadResponse{State: resp.State, Diagnostics: resp.Diagnostics})
}

func (r *TeamResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state teamResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	accountID := state.AccountID.ValueString()
	if state.AccountID.IsNull() || accountID == "" {
		accountID = r.client.GetConfig().DefaultHeader["X-Account-ID"]
	}
	apiResp, httpResp, err := r.client.TeamAPI.TeamsIdGet(ctx, state.ID.ValueString()).XAccountId(accountID).Execute()
	if err != nil {
		if httpResp != nil && httpResp.StatusCode == 404 {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Error reading team", err.Error())
		return
	}
	if apiResp.Data == nil {
		resp.State.RemoveResource(ctx)
		return
	}
	team := apiResp.Data
	state.Name = types.StringValue(team.GetName())
	state.AccountID = types.StringValue(team.GetAccountId())
	state.Description = types.StringValue(team.GetDescription())
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *TeamResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan teamResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	accountID := plan.AccountID.ValueString()
	if plan.AccountID.IsNull() || accountID == "" {
		accountID = r.client.GetConfig().DefaultHeader["X-Account-ID"]
	}
	payload := cs.ModelTeamRequestPayload{
		Name:        &[]string{plan.Name.ValueString()}[0],
		Description: &[]string{plan.Description.ValueString()}[0],
	}
	_, _, err := r.client.TeamAPI.TeamsIdPut(ctx, plan.ID.ValueString()).Body(payload).XAccountId(accountID).Execute()
	if err != nil {
		resp.Diagnostics.AddError("Error updating team", err.Error())
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
	r.Read(ctx, resource.ReadRequest{State: resp.State}, &resource.ReadResponse{State: resp.State, Diagnostics: resp.Diagnostics})
}

func (r *TeamResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state teamResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	accountID := state.AccountID.ValueString()
	if state.AccountID.IsNull() || accountID == "" {
		accountID = r.client.GetConfig().DefaultHeader["X-Account-ID"]
	}
	_, httpResp, err := r.client.TeamAPI.TeamsIdDelete(ctx, state.ID.ValueString()).XAccountId(accountID).Execute()
	if err != nil {
		if httpResp != nil && httpResp.StatusCode == 404 {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Error deleting team", err.Error())
		return
	}
	resp.State.RemoveResource(ctx)
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
