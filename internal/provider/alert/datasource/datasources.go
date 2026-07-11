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

type AlertsDataSource struct {
	client    *csv2.ClientWithResponses
	accountID string
}

var _ datasource.DataSource = &AlertsDataSource{}

func NewAlertsDataSource() datasource.DataSource {
	return &AlertsDataSource{}
}

func (d *AlertsDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = "computesphere_alerts"
}

func (d *AlertsDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = PluralSchema(ctx)
}

type alertItemModel struct {
	ID               types.String `tfsdk:"id"`
	ProjectID        types.String `tfsdk:"project_id"`
	EnvironmentID    types.String `tfsdk:"environment_id"`
	AlertType        types.String `tfsdk:"alert_type"`
	Severity         types.String `tfsdk:"severity"`
	Threshold        types.Int64  `tfsdk:"threshold"`
	EvaluationPeriod types.Int64  `tfsdk:"evaluation_period"`
	Active           types.Bool   `tfsdk:"active"`
}

type alertsDataSourceModel struct {
	ProjectID types.String     `tfsdk:"project_id"`
	Alerts    []alertItemModel `tfsdk:"alerts"`
}

func (d *AlertsDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	data := cstypes.ConfigureDatasource(req, resp)
	if data != nil {
		d.client = data.V2Client
		d.accountID = data.AccountID
	}
}

func (d *AlertsDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var state alertsDataSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	projectID, err := uuid.Parse(state.ProjectID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Invalid project_id", err.Error())
		return
	}

	apiResp, err := d.client.ListAlertRulesWithResponse(ctx, &csv2.ListAlertRulesParams{ProjectId: &projectID})
	if err != nil {
		resp.Diagnostics.AddError("Error listing alerts", err.Error())
		return
	}
	if apiResp.StatusCode() != http.StatusOK || apiResp.JSON200 == nil {
		resp.Diagnostics.AddError("Error listing alerts", cstypes.ProblemSummary(apiResp.Body, apiResp.StatusCode()))
		return
	}

	alerts := make([]alertItemModel, 0, len(apiResp.JSON200.Items))
	for _, a := range apiResp.JSON200.Items {
		alerts = append(alerts, alertItemModel{
			ID:               types.StringValue(a.Id.String()),
			ProjectID:        types.StringValue(a.ProjectId.String()),
			EnvironmentID:    types.StringValue(a.EnvironmentId.String()),
			AlertType:        types.StringValue(string(a.AlertType)),
			Severity:         types.StringValue(string(a.SeverityLevel)),
			Threshold:        types.Int64Value(int64(a.Threshold)),
			EvaluationPeriod: types.Int64Value(int64(a.EvaluationPeriod)),
			Active:           types.BoolValue(a.Active),
		})
	}
	state.Alerts = alerts
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}
