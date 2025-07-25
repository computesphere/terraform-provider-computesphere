package provider

import (
	"context"

	cs "github.com/computesphere/cli/cs"
	cstypes "github.com/computesphere/terraform-provider-computesphere/internal/provider/types"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/identityschema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

type AlertResource struct {
	client *cs.APIClient
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
	Name             types.String `tfsdk:"name"`
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
		r.client = data.Client
	}
}

func (r *AlertResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan alertResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	payload := cs.ModelAlertRuleRequest{
		ProjectId:     ptrString(plan.ProjectID.ValueString()),
		EnvironmentId: ptrString(plan.EnvironmentID.ValueString()),
		AlertType:     ptrString(plan.AlertType.ValueString()),
		SeverityLevel: ptrString(plan.Severity.ValueString()),
	}
	if !plan.Threshold.IsNull() {
		val := int(plan.Threshold.ValueInt64())
		payload.Threshold = &val
	}
	if !plan.EvaluationPeriod.IsNull() {
		val := int(plan.EvaluationPeriod.ValueInt64())
		payload.EvaluationPeriod = &val
	}
	if !plan.Active.IsNull() {
		val := plan.Active.ValueBool()
		payload.Active = &val
	}
	apiResp, _, err := r.client.AlertRuleAPI.AlertsPost(ctx).Body(payload).Execute()
	if err != nil {
		resp.Diagnostics.AddError("Error creating alert", err.Error())
		return
	}
	if apiResp == nil || apiResp.Data == nil {
		resp.Diagnostics.AddError("Alert creation failed", "No ID returned")
		return
	}
	plan.ID = types.StringValue(*apiResp.Data)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *AlertResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state alertResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	apiResp, httpResp, err := r.client.AlertRuleAPI.AlertsIdGet(ctx, state.ID.ValueString()).Execute()
	if err != nil {
		if httpResp != nil && httpResp.StatusCode == 404 {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Error reading alert", err.Error())
		return
	}
	if apiResp.Data == nil {
		resp.State.RemoveResource(ctx)
		return
	}
	alert := apiResp.Data
	state.ProjectID = types.StringValue(alert.GetProjectId())
	state.EnvironmentID = types.StringValue(alert.GetEnvironmentId())
	state.AlertType = types.StringValue(alert.GetAlertType())
	state.Severity = types.StringValue(alert.GetSeverityLevel())
	if alert.Threshold != nil {
		state.Threshold = types.Int64Value(int64(*alert.Threshold))
	} else {
		state.Threshold = types.Int64Null()
	}
	if alert.EvaluationPeriod != nil {
		state.EvaluationPeriod = types.Int64Value(int64(*alert.EvaluationPeriod))
	} else {
		state.EvaluationPeriod = types.Int64Null()
	}
	if alert.Active != nil {
		state.Active = types.BoolPointerValue(alert.Active)
	} else {
		state.Active = types.BoolNull()
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *AlertResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan alertResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	payload := cs.ModelAlertRuleUpdateRequest{}
	if !plan.Active.IsNull() {
		val := plan.Active.ValueBool()
		payload.Active = &val
	}
	if !plan.AlertType.IsNull() {
		str := plan.AlertType.ValueString()
		payload.AlertType = &str
	}
	if !plan.EnvironmentID.IsNull() {
		str := plan.EnvironmentID.ValueString()
		payload.EnvironmentId = &str
	}
	if !plan.EvaluationPeriod.IsNull() {
		val := int(plan.EvaluationPeriod.ValueInt64())
		payload.EvaluationPeriod = &val
	}
	if !plan.Severity.IsNull() {
		str := plan.Severity.ValueString()
		payload.SeverityLevel = &str
	}
	if !plan.Threshold.IsNull() {
		val := int(plan.Threshold.ValueInt64())
		payload.Threshold = &val
	}
	_, _, err := r.client.AlertRuleAPI.AlertsIdPut(ctx, plan.ID.ValueString()).Body(payload).Execute()
	if err != nil {
		resp.Diagnostics.AddError("Error updating alert", err.Error())
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *AlertResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state alertResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	_, httpResp, err := r.client.AlertRuleAPI.AlertsIdDelete(ctx, state.ID.ValueString()).Execute()
	if err != nil {
		if httpResp != nil && httpResp.StatusCode == 404 {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Error deleting alert", err.Error())
		return
	}
	resp.State.RemoveResource(ctx)
}

func ptrString(s string) *string { return &s }
