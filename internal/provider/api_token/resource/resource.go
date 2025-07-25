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

type ApiTokenResource struct {
	client *cs.APIClient
}

var _ resource.Resource = &ApiTokenResource{}
var _ resource.ResourceWithImportState = &ApiTokenResource{}
var _ resource.ResourceWithIdentity = &ApiTokenResource{}

func NewApiTokenResource() resource.Resource {
	return &ApiTokenResource{}
}

func (r *ApiTokenResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = "computesphere_api_token"
}

func (r *ApiTokenResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = Schema(ctx)
}

type apiTokenResourceModel struct {
	Name       types.String   `tfsdk:"name"`
	Scope      types.String   `tfsdk:"scope"`
	Expiry     types.String   `tfsdk:"expiry"`
	Type       types.String   `tfsdk:"type"`
	AccountID  types.String   `tfsdk:"account_id"`
	AccountIDs []types.String `tfsdk:"account_ids"`
	ProjectIDs []types.String `tfsdk:"project_ids"`
	ID         types.String   `tfsdk:"id"`
	Token      types.String   `tfsdk:"token"`
	CreatedAt  types.String   `tfsdk:"created_at"`
	UserID     types.String   `tfsdk:"user_id"`
	Accounts   []types.String `tfsdk:"accounts"`
	Projects   []types.String `tfsdk:"projects"`
}

func (r *ApiTokenResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	data := cstypes.ConfigureResource(req, resp)
	if data != nil {
		r.client = data.Client
	}
}

func (r *ApiTokenResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan apiTokenResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	payload := cs.ModelApiTokenRequestPayload{
		Name:  &[]string{plan.Name.ValueString()}[0],
		Scope: &[]string{plan.Scope.ValueString()}[0],
	}
	if !plan.Expiry.IsNull() {
		expiry := plan.Expiry.ValueString()
		payload.Expiry = &expiry
	}
	if !plan.Type.IsNull() {
		typeVal := plan.Type.ValueString()
		payload.Type = &typeVal
	}
	if !plan.AccountID.IsNull() {
		accountID := plan.AccountID.ValueString()
		payload.AccountId = &accountID
	}
	if plan.AccountIDs != nil {
		ids := []string{}
		for _, id := range plan.AccountIDs {
			if !id.IsNull() {
				ids = append(ids, id.ValueString())
			}
		}
		payload.AccountIds = ids
	}
	if plan.ProjectIDs != nil {
		ids := []string{}
		for _, id := range plan.ProjectIDs {
			if !id.IsNull() {
				ids = append(ids, id.ValueString())
			}
		}
		payload.ProjectIds = ids
	}
	apiResp, _, err := r.client.ApiTokenAPI.ApiTokensPost(ctx).Body(payload).Execute()
	if err != nil {
		resp.Diagnostics.AddError("Error creating API token", err.Error())
		return
	}
	if apiResp.Data == nil || apiResp.Data.Id == nil {
		resp.Diagnostics.AddError("API token creation failed", "No ID returned")
		return
	}
	plan.ID = types.StringValue(*apiResp.Data.Id)
	plan.Token = types.StringPointerValue(apiResp.Data.Token)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
	r.Read(ctx, resource.ReadRequest{State: resp.State}, &resource.ReadResponse{State: resp.State, Diagnostics: resp.Diagnostics})
}

func (r *ApiTokenResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state apiTokenResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	apiResp, httpResp, err := r.client.ApiTokenAPI.ApiTokensIdGet(ctx, state.ID.ValueString()).Execute()
	if err != nil {
		if httpResp != nil && httpResp.StatusCode == 404 {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Error reading API token", err.Error())
		return
	}
	if apiResp.Data == nil {
		resp.State.RemoveResource(ctx)
		return
	}
	token := apiResp.Data
	state.Name = types.StringValue(token.GetName())
	state.Scope = types.StringValue(token.GetScope())
	state.Expiry = types.StringPointerValue(token.Expiry)
	state.Type = types.StringPointerValue(token.Type)
	state.AccountID = types.StringPointerValue(token.AccountId)
	if token.AccountIds != nil {
		ids := make([]types.String, 0, len(token.AccountIds))
		for _, id := range token.AccountIds {
			ids = append(ids, types.StringValue(id))
		}
		state.AccountIDs = ids
	}
	if token.ProjectIds != nil {
		ids := make([]types.String, 0, len(token.ProjectIds))
		for _, id := range token.ProjectIds {
			ids = append(ids, types.StringValue(id))
		}
		state.ProjectIDs = ids
	}
	state.Token = types.StringPointerValue(token.Token)
	state.CreatedAt = types.StringValue(token.GetCreatedAt())
	state.UserID = types.StringPointerValue(token.UserId)
	if token.Accounts != nil {
		accounts := make([]types.String, 0, len(token.Accounts))
		for _, a := range token.Accounts {
			accounts = append(accounts, types.StringValue(a.GetId()))
		}
		state.Accounts = accounts
	}
	if token.Projects != nil {
		projects := make([]types.String, 0, len(token.Projects))
		for _, p := range token.Projects {
			projects = append(projects, types.StringValue(p.GetId()))
		}
		state.Projects = projects
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *ApiTokenResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan apiTokenResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
	r.Read(ctx, resource.ReadRequest{State: resp.State}, &resource.ReadResponse{State: resp.State, Diagnostics: resp.Diagnostics})
}

func (r *ApiTokenResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state apiTokenResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	_, httpResp, err := r.client.ApiTokenAPI.ApiTokensIdDelete(ctx, state.ID.ValueString()).Execute()
	if err != nil {
		if httpResp != nil && httpResp.StatusCode == 404 {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Error deleting API token", err.Error())
		return
	}
	resp.State.RemoveResource(ctx)
}

func (r *ApiTokenResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

func (r *ApiTokenResource) IdentitySchema(ctx context.Context, req resource.IdentitySchemaRequest, resp *resource.IdentitySchemaResponse) {
	resp.IdentitySchema = identityschema.Schema{
		Attributes: map[string]identityschema.Attribute{
			"id": identityschema.StringAttribute{RequiredForImport: true},
		},
	}
}
