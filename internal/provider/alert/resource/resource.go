package provider

import (
	"context"
	"net/http"

	csv2 "github.com/computesphere/computesphere-go"
	cstypes "github.com/computesphere/terraform-provider-computesphere/internal/provider/types"
	"github.com/google/uuid"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/identityschema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

type AlertResource struct {
	client    *csv2.ClientWithResponses
	accountID string
}

var _ resource.Resource = &AlertResource{}
var _ resource.ResourceWithIdentity = &AlertResource{}

func NewAlertResource() resource.Resource {
	return &AlertResource{}
}

func (r *AlertResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = "computesphere_alert"
}

func (r *AlertResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = Schema(ctx)
}

func (r *AlertResource) IdentitySchema(ctx context.Context, req resource.IdentitySchemaRequest, resp *resource.IdentitySchemaResponse) {
	resp.IdentitySchema = identityschema.Schema{
		Attributes: map[string]identityschema.Attribute{
			"id": identityschema.StringAttribute{RequiredForImport: true},
		},
	}
}

type alertResourceModel struct {
	ProjectID        types.String `tfsdk:"project_id"`
	EnvironmentID    types.String `tfsdk:"environment_id"`
	AlertType        types.String `tfsdk:"alert_type"`
	Severity         types.String `tfsdk:"severity"`
	Threshold        types.Int64  `tfsdk:"threshold"`
	EvaluationPeriod types.Int64  `tfsdk:"evaluation_period"`
	Active           types.Bool   `tfsdk:"active"`
	ID               types.String `tfsdk:"id"`
}

func (r *AlertResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	data := cstypes.ConfigureResource(req, resp)
	if data != nil {
		r.client = data.V2Client
		r.accountID = data.AccountID
	}
}

func (m *alertResourceModel) apply(a *csv2.AlertRule) {
	m.ID = types.StringValue(a.Id.String())
	m.ProjectID = types.StringValue(a.ProjectId.String())
	m.EnvironmentID = types.StringValue(a.EnvironmentId.String())
	m.AlertType = types.StringValue(string(a.AlertType))
	m.Severity = types.StringValue(string(a.SeverityLevel))
	m.Threshold = types.Int64Value(int64(a.Threshold))
	m.EvaluationPeriod = types.Int64Value(int64(a.EvaluationPeriod))
	m.Active = types.BoolValue(a.Active)
}

func (r *AlertResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan alertResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	accountID, err := uuid.Parse(r.accountID)
	if err != nil {
		resp.Diagnostics.AddError("Invalid account_id", err.Error())
		return
	}
	projectID, err := uuid.Parse(plan.ProjectID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Invalid project_id", err.Error())
		return
	}
	body := csv2.CreateAlertRuleRequest{
		AlertType:        csv2.CreateAlertRuleRequestAlertType(plan.AlertType.ValueString()),
		EvaluationPeriod: int(plan.EvaluationPeriod.ValueInt64()),
		ProjectId:        projectID,
		SeverityLevel:    csv2.CreateAlertRuleRequestSeverityLevel(plan.Severity.ValueString()),
		Threshold:        int(plan.Threshold.ValueInt64()),
	}
	if !plan.Active.IsNull() && !plan.Active.IsUnknown() {
		active := plan.Active.ValueBool()
		body.Active = &active
	}
	if envID, perr := uuid.Parse(plan.EnvironmentID.ValueString()); perr == nil {
		body.EnvironmentId = &envID
	}

	apiResp, err := r.client.CreateAlertRuleWithResponse(ctx, &csv2.CreateAlertRuleParams{XAccountId: accountID}, body)
	if err != nil {
		resp.Diagnostics.AddError("Error creating alert", err.Error())
		return
	}
	if apiResp.StatusCode() != http.StatusCreated || apiResp.JSON201 == nil {
		resp.Diagnostics.AddError("Error creating alert", cstypes.ProblemSummary(apiResp.Body, apiResp.StatusCode()))
		return
	}

	var state alertResourceModel
	state.apply(apiResp.JSON201)
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *AlertResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state alertResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	arid, err := uuid.Parse(state.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Invalid alert id", err.Error())
		return
	}

	apiResp, err := r.client.GetAlertRuleWithResponse(ctx, csv2.AlertRuleId(arid))
	if err != nil {
		resp.Diagnostics.AddError("Error reading alert", err.Error())
		return
	}
	if apiResp.StatusCode() == http.StatusNotFound {
		resp.State.RemoveResource(ctx)
		return
	}
	if apiResp.StatusCode() != http.StatusOK || apiResp.JSON200 == nil {
		resp.Diagnostics.AddError("Error reading alert", cstypes.ProblemSummary(apiResp.Body, apiResp.StatusCode()))
		return
	}

	state.apply(apiResp.JSON200)
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *AlertResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan alertResourceModel
	var state alertResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	arid, err := uuid.Parse(state.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Invalid alert id", err.Error())
		return
	}

	alertType := csv2.UpdateAlertRuleRequestAlertType(plan.AlertType.ValueString())
	severity := csv2.UpdateAlertRuleRequestSeverityLevel(plan.Severity.ValueString())
	evalPeriod := int(plan.EvaluationPeriod.ValueInt64())
	threshold := int(plan.Threshold.ValueInt64())
	body := csv2.UpdateAlertRuleRequest{
		AlertType:        &alertType,
		SeverityLevel:    &severity,
		EvaluationPeriod: &evalPeriod,
		Threshold:        &threshold,
	}
	if !plan.Active.IsNull() && !plan.Active.IsUnknown() {
		active := plan.Active.ValueBool()
		body.Active = &active
	}
	if envID, perr := uuid.Parse(plan.EnvironmentID.ValueString()); perr == nil {
		body.EnvironmentId = &envID
	}

	apiResp, err := r.client.UpdateAlertRuleWithResponse(ctx, csv2.AlertRuleId(arid), body)
	if err != nil {
		resp.Diagnostics.AddError("Error updating alert", err.Error())
		return
	}
	if apiResp.StatusCode() != http.StatusOK || apiResp.JSON200 == nil {
		resp.Diagnostics.AddError("Error updating alert", cstypes.ProblemSummary(apiResp.Body, apiResp.StatusCode()))
		return
	}

	var newState alertResourceModel
	newState.apply(apiResp.JSON200)
	resp.Diagnostics.Append(resp.State.Set(ctx, &newState)...)
}

func (r *AlertResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state alertResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	arid, err := uuid.Parse(state.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Invalid alert id", err.Error())
		return
	}

	apiResp, err := r.client.DeleteAlertRuleWithResponse(ctx, csv2.AlertRuleId(arid))
	if err != nil {
		resp.Diagnostics.AddError("Error deleting alert", err.Error())
		return
	}
	switch apiResp.StatusCode() {
	case http.StatusOK, http.StatusNoContent, http.StatusAccepted, http.StatusNotFound:
		resp.State.RemoveResource(ctx)
	default:
		resp.Diagnostics.AddError("Error deleting alert", cstypes.ProblemSummary(apiResp.Body, apiResp.StatusCode()))
	}
}
