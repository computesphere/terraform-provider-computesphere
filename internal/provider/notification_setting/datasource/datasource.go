package provider

import (
	"context"
	"net/http"

	csv2 "github.com/computesphere/computesphere-go"
	cstypes "github.com/computesphere/terraform-provider-computesphere/internal/provider/types"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

type NotificationSettingDataSource struct {
	client    *csv2.ClientWithResponses
	accountID string
}

var _ datasource.DataSource = &NotificationSettingDataSource{}

func NewNotificationSettingDataSource() datasource.DataSource {
	return &NotificationSettingDataSource{}
}

func (d *NotificationSettingDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = "computesphere_notification_setting"
}

func (d *NotificationSettingDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = Schema(ctx)
}

type notificationSettingDataSourceModel struct {
	Activity       types.Bool     `tfsdk:"activity"`
	Billing        types.Bool     `tfsdk:"billing"`
	Deployment     types.Bool     `tfsdk:"deployment"`
	EmailEnabled   types.Bool     `tfsdk:"email_enabled"`
	Emails         []types.String `tfsdk:"emails"`
	InappEnabled   types.Bool     `tfsdk:"inapp_enabled"`
	Invites        types.Bool     `tfsdk:"invites"`
	Payment        types.Bool     `tfsdk:"payment"`
	WebhookEnabled types.Bool     `tfsdk:"webhook_enabled"`
	ID             types.String   `tfsdk:"id"`
	UserID         types.String   `tfsdk:"user_id"`
}

func (d *NotificationSettingDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	data := cstypes.ConfigureDatasource(req, resp)
	if data != nil {
		d.client = data.V2Client
		d.accountID = data.AccountID
	}
}

func (d *NotificationSettingDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var state notificationSettingDataSourceModel
	apiResp, err := d.client.GetNotificationSettingsWithResponse(ctx)
	if err != nil {
		resp.Diagnostics.AddError("Error reading notification setting", err.Error())
		return
	}
	if apiResp.StatusCode() == http.StatusNotFound {
		resp.State.RemoveResource(ctx)
		return
	}
	if apiResp.StatusCode() != http.StatusOK || apiResp.JSON200 == nil {
		resp.Diagnostics.AddError("Error reading notification setting", cstypes.ProblemSummary(apiResp.Body, apiResp.StatusCode()))
		return
	}
	n := apiResp.JSON200
	state.ID = types.StringValue(n.Id.String())
	state.UserID = types.StringValue(n.UserId.String())
	state.Activity = types.BoolValue(n.Activity)
	state.Billing = types.BoolValue(n.Billing)
	state.Deployment = types.BoolValue(n.Deployment)
	state.EmailEnabled = types.BoolValue(n.EmailEnabled)
	state.InappEnabled = types.BoolValue(n.InappEnabled)
	state.Invites = types.BoolValue(n.Invites)
	state.Payment = types.BoolValue(n.Payment)
	state.WebhookEnabled = types.BoolValue(n.WebhookEnabled)
	emails := make([]types.String, 0, len(n.Emails))
	for _, e := range n.Emails {
		emails = append(emails, types.StringValue(string(e)))
	}
	state.Emails = emails
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}
