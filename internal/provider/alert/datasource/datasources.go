package provider

import (
	"context"

	cs "github.com/computesphere/cli/cs"
	cstypes "github.com/computesphere/terraform-provider-computesphere/internal/provider/types"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

type AlertsDataSource struct {
	client *cs.APIClient
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
	Alerts []alertItemModel `tfsdk:"alerts"`
}

func (d *AlertsDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	data := cstypes.ConfigureDatasource(req, resp)
	if data != nil {
		d.client = data.Client
	}
}

func (d *AlertsDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var state alertsDataSourceModel
	apiResp, _, err := d.client.AlertRuleAPI.AlertsGet(ctx).Execute()
	if err != nil {
		resp.Diagnostics.AddError("Error listing alerts", err.Error())
		return
	}
	if apiResp.Data == nil {
		resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
		return
	}
	alerts := make([]alertItemModel, 0, len(apiResp.Data))
	for _, a := range apiResp.Data {
		item := alertItemModel{
			ID:            types.StringValue(a.GetId()),
			ProjectID:     types.StringValue(a.GetProjectId()),
			EnvironmentID: types.StringValue(a.GetEnvironmentId()),
			AlertType:     types.StringValue(a.GetAlertType()),
			Severity:      types.StringValue(a.GetSeverityLevel()),
		}
		if a.Threshold != nil {
			item.Threshold = types.Int64Value(int64(*a.Threshold))
		} else {
			item.Threshold = types.Int64Null()
		}
		if a.EvaluationPeriod != nil {
			item.EvaluationPeriod = types.Int64Value(int64(*a.EvaluationPeriod))
		} else {
			item.EvaluationPeriod = types.Int64Null()
		}
		if a.Active != nil {
			item.Active = types.BoolPointerValue(a.Active)
		} else {
			item.Active = types.BoolNull()
		}
		alerts = append(alerts, item)
	}
	state.Alerts = alerts
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}
