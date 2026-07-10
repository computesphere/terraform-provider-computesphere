package provider

import (
	"context"
	"encoding/json"
	"net/http"

	csv2 "github.com/computesphere/computesphere-go"
	cstypes "github.com/computesphere/terraform-provider-computesphere/internal/provider/types"
	"github.com/google/uuid"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/identityschema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

type GuardrailResource struct {
	client    *csv2.ClientWithResponses
	accountID string
}

var _ resource.Resource = &GuardrailResource{}
var _ resource.ResourceWithIdentity = &GuardrailResource{}

func NewGuardrailResource() resource.Resource {
	return &GuardrailResource{}
}

func (r *GuardrailResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = "computesphere_guardrail"
}

func (r *GuardrailResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = Schema(ctx)
}

func (r *GuardrailResource) IdentitySchema(ctx context.Context, req resource.IdentitySchemaRequest, resp *resource.IdentitySchemaResponse) {
	resp.IdentitySchema = identityschema.Schema{
		Attributes: map[string]identityschema.Attribute{
			"id": identityschema.StringAttribute{RequiredForImport: true},
		},
	}
}

// guardrailResourceModel — the nested `rules` collection is intentionally not
// modeled here; the v2 create/update surface takes rule ids and the response
// shape is a separate follow-up.
type guardrailResourceModel struct {
	Name                 types.String `tfsdk:"name"`
	Description          types.String `tfsdk:"description"`
	Effect               types.String `tfsdk:"effect"`
	Message              types.String `tfsdk:"message"`
	Scope                types.String `tfsdk:"scope"`
	Status               types.Bool   `tfsdk:"status"`
	Type                 types.String `tfsdk:"type"`
	ID                   types.String `tfsdk:"id"`
	AccountID            types.String `tfsdk:"account_id"`
	CreatedBy            types.String `tfsdk:"created_by"`
	IsPredefinedAssigned types.Bool   `tfsdk:"is_predefined_assigned"`
}

func (r *GuardrailResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	data := cstypes.ConfigureResource(req, resp)
	if data != nil {
		r.client = data.V2Client
		r.accountID = data.AccountID
	}
}

func (m *guardrailResourceModel) apply(g *csv2.Guardrail) {
	m.ID = types.StringValue(g.Id.String())
	m.Name = types.StringValue(g.Name)
	m.Description = types.StringValue(g.Description)
	m.Effect = types.StringValue(string(g.Effect))
	m.Message = types.StringValue(g.Message)
	m.Scope = types.StringValue(string(g.Scope))
	m.Status = types.BoolValue(g.Status)
	m.Type = types.StringValue(string(g.Type))
	m.AccountID = types.StringValue(g.AccountId.String())
	m.CreatedBy = types.StringValue(g.CreatedBy.String())
	m.IsPredefinedAssigned = types.BoolValue(g.IsPredefinedAssigned)
}

func (m *guardrailResourceModel) toRequest() csv2.CreateGuardrailRequest {
	return csv2.CreateGuardrailRequest{
		Description: m.Description.ValueString(),
		Effect:      csv2.CreateGuardrailRequestEffect(m.Effect.ValueString()),
		Message:     cstypes.StringPtr(m.Message.ValueString()),
		Name:        m.Name.ValueString(),
		Scope:       csv2.CreateGuardrailRequestScope(m.Scope.ValueString()),
		Status:      m.Status.ValueBool(),
		Type:        csv2.CreateGuardrailRequestType(m.Type.ValueString()),
	}
}

func (r *GuardrailResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan guardrailResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	accountID, err := uuid.Parse(r.accountID)
	if err != nil {
		resp.Diagnostics.AddError("Invalid account_id", err.Error())
		return
	}

	apiResp, err := r.client.CreateGuardrailWithResponse(ctx, &csv2.CreateGuardrailParams{XAccountId: accountID}, plan.toRequest())
	if err != nil {
		resp.Diagnostics.AddError("Error creating guardrail", err.Error())
		return
	}
	if apiResp.StatusCode() != http.StatusCreated && apiResp.StatusCode() != http.StatusOK {
		resp.Diagnostics.AddError("Error creating guardrail", cstypes.ProblemSummary(apiResp.Body, apiResp.StatusCode()))
		return
	}
	// The create response is not typed by the SDK; decode the guardrail from the
	// raw body to recover its id, then set full state.
	var created csv2.Guardrail
	if err := json.Unmarshal(apiResp.Body, &created); err != nil || created.Id == uuid.Nil {
		resp.Diagnostics.AddError("Error creating guardrail", "could not read the created guardrail from the API response")
		return
	}

	var state guardrailResourceModel
	state.apply(&created)
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *GuardrailResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state guardrailResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	accountID, err := uuid.Parse(r.accountID)
	if err != nil {
		resp.Diagnostics.AddError("Invalid account_id", err.Error())
		return
	}
	gid, err := uuid.Parse(state.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Invalid guardrail id", err.Error())
		return
	}

	apiResp, err := r.client.GetGuardrailWithResponse(ctx, csv2.GuardrailId(gid), &csv2.GetGuardrailParams{XAccountId: accountID})
	if err != nil {
		resp.Diagnostics.AddError("Error reading guardrail", err.Error())
		return
	}
	if apiResp.StatusCode() == http.StatusNotFound {
		resp.State.RemoveResource(ctx)
		return
	}
	if apiResp.StatusCode() != http.StatusOK || apiResp.JSON200 == nil {
		resp.Diagnostics.AddError("Error reading guardrail", cstypes.ProblemSummary(apiResp.Body, apiResp.StatusCode()))
		return
	}

	state.apply(apiResp.JSON200)
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *GuardrailResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan guardrailResourceModel
	var state guardrailResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	accountID, err := uuid.Parse(r.accountID)
	if err != nil {
		resp.Diagnostics.AddError("Invalid account_id", err.Error())
		return
	}
	gid, err := uuid.Parse(state.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Invalid guardrail id", err.Error())
		return
	}

	apiResp, err := r.client.UpdateGuardrailWithResponse(ctx, csv2.GuardrailId(gid), &csv2.UpdateGuardrailParams{XAccountId: accountID}, plan.toRequest())
	if err != nil {
		resp.Diagnostics.AddError("Error updating guardrail", err.Error())
		return
	}
	if apiResp.StatusCode() != http.StatusOK && apiResp.StatusCode() != http.StatusNoContent {
		resp.Diagnostics.AddError("Error updating guardrail", cstypes.ProblemSummary(apiResp.Body, apiResp.StatusCode()))
		return
	}

	// The update response is not typed; re-read to refresh state.
	getResp, err := r.client.GetGuardrailWithResponse(ctx, csv2.GuardrailId(gid), &csv2.GetGuardrailParams{XAccountId: accountID})
	if err != nil || getResp.StatusCode() != http.StatusOK || getResp.JSON200 == nil {
		resp.Diagnostics.AddError("Error updating guardrail", "guardrail updated but could not be re-read")
		return
	}
	plan.apply(getResp.JSON200)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *GuardrailResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state guardrailResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	accountID, err := uuid.Parse(r.accountID)
	if err != nil {
		resp.Diagnostics.AddError("Invalid account_id", err.Error())
		return
	}
	gid, err := uuid.Parse(state.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Invalid guardrail id", err.Error())
		return
	}

	apiResp, err := r.client.DeleteGuardrailWithResponse(ctx, csv2.GuardrailId(gid), &csv2.DeleteGuardrailParams{XAccountId: accountID})
	if err != nil {
		resp.Diagnostics.AddError("Error deleting guardrail", err.Error())
		return
	}
	switch apiResp.StatusCode() {
	case http.StatusOK, http.StatusNoContent, http.StatusAccepted, http.StatusNotFound:
		resp.State.RemoveResource(ctx)
	default:
		resp.Diagnostics.AddError("Error deleting guardrail", cstypes.ProblemSummary(apiResp.Body, apiResp.StatusCode()))
	}
}
