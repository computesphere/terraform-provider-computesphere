package provider

import (
	"context"
	"net/http"
	"time"

	csv2 "github.com/computesphere/computesphere-go"
	cstypes "github.com/computesphere/terraform-provider-computesphere/internal/provider/types"
	"github.com/google/uuid"
	openapi_types "github.com/oapi-codegen/runtime/types"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/identityschema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

type ApiTokenResource struct {
	client    *csv2.ClientWithResponses
	accountID string
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

// apiTokenResourceModel drops the v1-only type/accounts/projects attributes; the
// v2 token exposes account_ids/project_ids (uuid lists) and a single owning
// account_id.
type apiTokenResourceModel struct {
	Name       types.String   `tfsdk:"name"`
	Scope      types.String   `tfsdk:"scope"`
	Expiry     types.String   `tfsdk:"expiry"`
	AccountIDs []types.String `tfsdk:"account_ids"`
	ProjectIDs []types.String `tfsdk:"project_ids"`
	ID         types.String   `tfsdk:"id"`
	Token      types.String   `tfsdk:"token"`
	AccountID  types.String   `tfsdk:"account_id"`
	CreatedAt  types.String   `tfsdk:"created_at"`
	UserID     types.String   `tfsdk:"user_id"`
}

func (r *ApiTokenResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	data := cstypes.ConfigureResource(req, resp)
	if data != nil {
		r.client = data.V2Client
		r.accountID = data.AccountID
	}
}

func uuidStrings(ids []openapi_types.UUID) []types.String {
	out := make([]types.String, 0, len(ids))
	for _, id := range ids {
		out = append(out, types.StringValue(id.String()))
	}
	return out
}

// apply maps the token metadata into state. It never sets Token: the secret is
// only returned once (at creation) and cannot be re-read.
func (m *apiTokenResourceModel) apply(t *csv2.APIToken) {
	m.ID = types.StringValue(t.Id.String())
	m.Name = types.StringValue(t.Name)
	m.Scope = types.StringValue(string(t.Scope))
	m.Expiry = types.StringValue(t.Expiry.Format(time.RFC3339))
	m.AccountID = types.StringValue(t.AccountId)
	m.AccountIDs = uuidStrings(t.AccountIds)
	m.ProjectIDs = uuidStrings(t.ProjectIds)
	m.CreatedAt = types.StringValue(t.CreatedAt.Format(time.RFC3339))
	m.UserID = types.StringValue(t.UserId.String())
}

func parseUUIDList(in []types.String) (*[]openapi_types.UUID, error) {
	if len(in) == 0 {
		return nil, nil
	}
	out := make([]openapi_types.UUID, 0, len(in))
	for _, s := range in {
		id, err := uuid.Parse(s.ValueString())
		if err != nil {
			return nil, err
		}
		out = append(out, id)
	}
	return &out, nil
}

func (r *ApiTokenResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan apiTokenResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	accountID, err := uuid.Parse(r.accountID)
	if err != nil {
		resp.Diagnostics.AddError("Invalid account_id", err.Error())
		return
	}
	expiry, err := time.Parse(time.RFC3339, plan.Expiry.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Invalid expiry", "expiry must be an RFC3339 timestamp: "+err.Error())
		return
	}
	accountIDs, err := parseUUIDList(plan.AccountIDs)
	if err != nil {
		resp.Diagnostics.AddError("Invalid account_ids", err.Error())
		return
	}
	projectIDs, err := parseUUIDList(plan.ProjectIDs)
	if err != nil {
		resp.Diagnostics.AddError("Invalid project_ids", err.Error())
		return
	}

	apiResp, err := r.client.CreateAPITokenWithResponse(ctx, &csv2.CreateAPITokenParams{XAccountId: accountID}, csv2.CreateAPITokenRequest{
		AccountIds: accountIDs,
		Expiry:     expiry,
		Name:       plan.Name.ValueString(),
		ProjectIds: projectIDs,
		Scope:      csv2.CreateAPITokenRequestScope(plan.Scope.ValueString()),
	})
	if err != nil {
		resp.Diagnostics.AddError("Error creating API token", err.Error())
		return
	}
	if apiResp.StatusCode() != http.StatusCreated || apiResp.JSON201 == nil {
		resp.Diagnostics.AddError("Error creating API token", cstypes.ProblemSummary(apiResp.Body, apiResp.StatusCode()))
		return
	}

	var state apiTokenResourceModel
	state.apply(&apiResp.JSON201.Token)
	state.Token = types.StringValue(apiResp.JSON201.Secret)
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *ApiTokenResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state apiTokenResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	tid, err := uuid.Parse(state.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Invalid api token id", err.Error())
		return
	}

	apiResp, err := r.client.GetAPITokenWithResponse(ctx, tid)
	if err != nil {
		resp.Diagnostics.AddError("Error reading API token", err.Error())
		return
	}
	if apiResp.StatusCode() == http.StatusNotFound {
		resp.State.RemoveResource(ctx)
		return
	}
	if apiResp.StatusCode() != http.StatusOK || apiResp.JSON200 == nil {
		resp.Diagnostics.AddError("Error reading API token", cstypes.ProblemSummary(apiResp.Body, apiResp.StatusCode()))
		return
	}

	// Preserve the secret from prior state; Get does not return it.
	token := state.Token
	state.apply(apiResp.JSON200)
	state.Token = token
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *ApiTokenResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	// API tokens are immutable (every mutable attribute forces replacement), so
	// update only carries state forward.
	var plan apiTokenResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *ApiTokenResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state apiTokenResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	tid, err := uuid.Parse(state.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Invalid api token id", err.Error())
		return
	}

	apiResp, err := r.client.RevokeAPITokenWithResponse(ctx, tid)
	if err != nil {
		resp.Diagnostics.AddError("Error revoking API token", err.Error())
		return
	}
	switch apiResp.StatusCode() {
	case http.StatusOK, http.StatusNoContent, http.StatusAccepted, http.StatusNotFound:
		resp.State.RemoveResource(ctx)
	default:
		resp.Diagnostics.AddError("Error revoking API token", cstypes.ProblemSummary(apiResp.Body, apiResp.StatusCode()))
	}
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
