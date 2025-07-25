package provider

import (
	"context"

	cs "github.com/computesphere/cli/cs"
	cstypes "github.com/computesphere/terraform-provider-computesphere/internal/provider/types"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

type NotificationSettingDataSource struct {
	client *cs.APIClient
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
	Activity       types.Bool                `tfsdk:"activity"`
	Billing        types.Bool                `tfsdk:"billing"`
	Deployment     types.Bool                `tfsdk:"deployment"`
	EmailEnabled   types.Bool                `tfsdk:"email_enabled"`
	Emails         []types.String            `tfsdk:"emails"`
	InappEnabled   types.Bool                `tfsdk:"inapp_enabled"`
	Invites        types.Bool                `tfsdk:"invites"`
	Payment        types.Bool                `tfsdk:"payment"`
	WebhookEnabled types.Bool                `tfsdk:"webhook_enabled"`
	Webhooks       []map[string]types.String `tfsdk:"webhooks"`
	ID             types.String              `tfsdk:"id"`
	UserID         types.String              `tfsdk:"user_id"`
}

func (d *NotificationSettingDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	data := cstypes.ConfigureDatasource(req, resp)
	if data != nil {
		d.client = data.Client
	}
}

func (d *NotificationSettingDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var state notificationSettingDataSourceModel
	client := d.client
	apiResp, httpResp, err := client.NotificationAPI.NotificationsSettingsGet(ctx).Execute()
	if err != nil {
		if httpResp != nil && httpResp.StatusCode == 404 {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Error reading notification setting", err.Error())
		return
	}
	if apiResp.Data == nil || apiResp.Data.Id == nil {
		resp.State.RemoveResource(ctx)
		return
	}
	state.ID = types.StringValue(apiResp.Data.GetId())
	if apiResp.Data.UserId != nil {
		state.UserID = types.StringPointerValue(apiResp.Data.UserId)
	} else {
		state.UserID = types.StringNull()
	}
	if apiResp.Data.Activity != nil {
		state.Activity = types.BoolPointerValue(apiResp.Data.Activity)
	} else {
		state.Activity = types.BoolNull()
	}
	if apiResp.Data.Billing != nil {
		state.Billing = types.BoolPointerValue(apiResp.Data.Billing)
	} else {
		state.Billing = types.BoolNull()
	}
	if apiResp.Data.Deployment != nil {
		state.Deployment = types.BoolPointerValue(apiResp.Data.Deployment)
	} else {
		state.Deployment = types.BoolNull()
	}
	if apiResp.Data.EmailEnabled != nil {
		state.EmailEnabled = types.BoolPointerValue(apiResp.Data.EmailEnabled)
	} else {
		state.EmailEnabled = types.BoolNull()
	}
	if apiResp.Data.Emails != nil {
		emails := make([]types.String, 0, len(apiResp.Data.Emails))
		for _, e := range apiResp.Data.Emails {
			emails = append(emails, types.StringValue(e))
		}
		state.Emails = emails
	}
	if apiResp.Data.InappEnabled != nil {
		state.InappEnabled = types.BoolPointerValue(apiResp.Data.InappEnabled)
	} else {
		state.InappEnabled = types.BoolNull()
	}
	if apiResp.Data.Invites != nil {
		state.Invites = types.BoolPointerValue(apiResp.Data.Invites)
	} else {
		state.Invites = types.BoolNull()
	}
	if apiResp.Data.Payment != nil {
		state.Payment = types.BoolPointerValue(apiResp.Data.Payment)
	} else {
		state.Payment = types.BoolNull()
	}
	if apiResp.Data.WebhookEnabled != nil {
		state.WebhookEnabled = types.BoolPointerValue(apiResp.Data.WebhookEnabled)
	} else {
		state.WebhookEnabled = types.BoolNull()
	}
	// Webhooks omitted for brevity; add as needed
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}
