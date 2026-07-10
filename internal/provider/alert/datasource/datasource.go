package provider

import (
	"context"
	"net/http"

	csv2 "github.com/computesphere/computesphere-go"
	cstypes "github.com/computesphere/terraform-provider-computesphere/internal/provider/types"
	"github.com/google/uuid"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

type AlertDataSource struct {
	client    *csv2.ClientWithResponses
	accountID string
}

var _ datasource.DataSource = &AlertDataSource{}

func NewAlertDataSource() datasource.DataSource {
	return &AlertDataSource{}
}

func (d *AlertDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = "computesphere_alert"
}

func (d *AlertDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = Schema(ctx)
}

type alertDataSourceModel struct {
	ID               types.String `tfsdk:"id"`
	ProjectID        types.String `tfsdk:"project_id"`
	EnvironmentID    types.String `tfsdk:"environment_id"`
	AlertType        types.String `tfsdk:"alert_type"`
	Severity         types.String `tfsdk:"severity"`
	Threshold        types.Int64  `tfsdk:"threshold"`
	EvaluationPeriod types.Int64  `tfsdk:"evaluation_period"`
	Active           types.Bool   `tfsdk:"active"`
}

func (d *AlertDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	data := cstypes.ConfigureDatasource(req, resp)
	if data != nil {
		d.client = data.V2Client
		d.accountID = data.AccountID
	}
}

func (d *AlertDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var state alertDataSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	arid, err := uuid.Parse(state.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Invalid alert id", err.Error())
		return
	}

	apiResp, err := d.client.GetAlertRuleWithResponse(ctx, arid)
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

	a := apiResp.JSON200
	state.ProjectID = types.StringValue(a.ProjectId.String())
	state.EnvironmentID = types.StringValue(a.EnvironmentId.String())
	state.AlertType = types.StringValue(string(a.AlertType))
	state.Severity = types.StringValue(string(a.SeverityLevel))
	state.Threshold = types.Int64Value(int64(a.Threshold))
	state.EvaluationPeriod = types.Int64Value(int64(a.EvaluationPeriod))
	state.Active = types.BoolValue(a.Active)
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}
