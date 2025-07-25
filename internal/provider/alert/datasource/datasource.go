package provider

import (
	"context"

	cs "github.com/computesphere/cli/cs"
	cstypes "github.com/computesphere/terraform-provider-computesphere/internal/provider/types"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

type AlertDataSource struct {
	client *cs.APIClient
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
		d.client = data.Client
	}
}

func (d *AlertDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var state alertDataSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	apiResp, httpResp, err := d.client.AlertRuleAPI.AlertsIdGet(ctx, state.ID.ValueString()).Execute()
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
