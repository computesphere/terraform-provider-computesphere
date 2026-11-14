package provider

import (
	"context"

	shared "github.com/computesphere/terraform-provider-computesphere/internal/provider/types/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
)

func Schema(ctx context.Context) schema.Schema {
	return schema.Schema{
		Description: "Manages the notification settings of the person who owns the API token: which topics are " +
			"delivered, and by in-app, email or webhook. Needs a full-access API token: notification settings span every " +
			"account the person is in, so a token scoped to an account or a project is refused.",
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
			"webhooks": schema.ListNestedAttribute{
				Optional: true, Sensitive: true,
				Description: "Webhooks that receive the notifications. Leave it out to keep the webhooks set in " +
					"the console as they are; set it to manage them here (an empty list removes them all).",
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"url": schema.StringAttribute{
							Required: true, Description: "Webhook URL. It must start with `https://`.",
						},
						"content_type": schema.StringAttribute{
							Optional: true, Computed: true,
							Description: "Body format: `json` (the default) or `text`.",
						},
						"token": schema.StringAttribute{
							Optional: true,
							Description: "Sent as the webhook's `Authorization` header. The API never returns it: leave " +
								"it out to keep the token already stored for this URL. A stored token can't be removed from " +
								"Terraform: change the URL, or remove the token in the console.",
						},
					},
				},
			},
			"id":      shared.NotificationSettingID,
			"user_id": shared.NotificationSettingUserID,
		},
	}
}
