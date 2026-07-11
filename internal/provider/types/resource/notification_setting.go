package resource

import (
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var NotificationSettingID = schema.StringAttribute{
	Computed:    true,
	Description: "Unique identifier for the notification setting.",
}

var NotificationSettingUserID = schema.StringAttribute{
	Computed:    true,
	Description: "User ID associated with the notification setting.",
}

var NotificationSettingActivity = schema.BoolAttribute{
	Optional:    true,
	Computed:    true,
	Description: "Activity notification enabled.",
}

var NotificationSettingBilling = schema.BoolAttribute{
	Optional:    true,
	Computed:    true,
	Description: "Billing notification enabled.",
}

var NotificationSettingDeployment = schema.BoolAttribute{
	Optional:    true,
	Computed:    true,
	Description: "Deployment notification enabled.",
}

var NotificationSettingEmailEnabled = schema.BoolAttribute{
	Optional:    true,
	Computed:    true,
	Description: "Email notifications enabled.",
}

var NotificationSettingEmails = schema.ListAttribute{
	ElementType: types.StringType,
	Optional:    true,
	Computed:    true,
	Description: "List of email addresses for notifications.",
}

var NotificationSettingInappEnabled = schema.BoolAttribute{
	Optional:    true,
	Computed:    true,
	Description: "In-app notifications enabled.",
}

var NotificationSettingInvites = schema.BoolAttribute{
	Optional:    true,
	Computed:    true,
	Description: "Invites notification enabled.",
}

var NotificationSettingPayment = schema.BoolAttribute{
	Optional:    true,
	Computed:    true,
	Description: "Payment notification enabled.",
}

var NotificationSettingWebhookEnabled = schema.BoolAttribute{
	Optional:    true,
	Computed:    true,
	Description: "Webhook notifications enabled.",
}

var NotificationSettingWebhooks = schema.ListAttribute{
	ElementType: types.MapType{ElemType: types.StringType},
	Optional:    true,
	Computed:    true,
	Description: "List of webhook configurations.",
}
