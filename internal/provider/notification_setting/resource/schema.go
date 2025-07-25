package provider

import (
	"context"

	shared "github.com/computesphere/terraform-provider-computesphere/internal/provider/types/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
)

func Schema(ctx context.Context) schema.Schema {
	return schema.Schema{
		Description: "Manages a notification setting resource.",
		Attributes: map[string]schema.Attribute{
			"activity":        shared.NotificationSettingActivity,
			"billing":         shared.NotificationSettingBilling,
			"deployment":      shared.NotificationSettingDeployment,
			"email_enabled":   shared.NotificationSettingEmailEnabled,
			"emails":          shared.NotificationSettingEmails,
			"inapp_enabled":   shared.NotificationSettingInappEnabled,
			"invites":         shared.NotificationSettingInvites,
			"payment":         shared.NotificationSettingPayment,
			"webhook_enabled": shared.NotificationSettingWebhookEnabled,
			"webhooks":        shared.NotificationSettingWebhooks,
			"id":              shared.NotificationSettingID,
			"user_id":         shared.NotificationSettingUserID,
		},
	}
}
