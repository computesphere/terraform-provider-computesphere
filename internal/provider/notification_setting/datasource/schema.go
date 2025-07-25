package provider

import (
	"context"

	shared "github.com/computesphere/terraform-provider-computesphere/internal/provider/types/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

func Schema(ctx context.Context) schema.Schema {
	return schema.Schema{
		Description: "Provides details for a notification setting.",
		Attributes: map[string]schema.Attribute{
			"activity":        schema.BoolAttribute{Computed: true, Description: shared.NotificationSettingActivity.Description},
			"billing":         schema.BoolAttribute{Computed: true, Description: shared.NotificationSettingBilling.Description},
			"deployment":      schema.BoolAttribute{Computed: true, Description: shared.NotificationSettingDeployment.Description},
			"email_enabled":   schema.BoolAttribute{Computed: true, Description: shared.NotificationSettingEmailEnabled.Description},
			"emails":          schema.ListAttribute{ElementType: types.StringType, Computed: true, Description: shared.NotificationSettingEmails.Description},
			"inapp_enabled":   schema.BoolAttribute{Computed: true, Description: shared.NotificationSettingInappEnabled.Description},
			"invites":         schema.BoolAttribute{Computed: true, Description: shared.NotificationSettingInvites.Description},
			"payment":         schema.BoolAttribute{Computed: true, Description: shared.NotificationSettingPayment.Description},
			"webhook_enabled": schema.BoolAttribute{Computed: true, Description: shared.NotificationSettingWebhookEnabled.Description},
			"webhooks":        schema.ListAttribute{ElementType: types.MapType{ElemType: types.StringType}, Computed: true, Description: shared.NotificationSettingWebhooks.Description},
			"id":              schema.StringAttribute{Computed: true, Description: shared.NotificationSettingID.Description},
			"user_id":         schema.StringAttribute{Computed: true, Description: shared.NotificationSettingUserID.Description},
		},
	}
}
