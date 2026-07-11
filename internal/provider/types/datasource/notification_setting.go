package datasource

import (
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
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
	Computed:    true,
	Description: "Activity notification enabled.",
}

var NotificationSettingBilling = schema.BoolAttribute{
	Computed:    true,
	Description: "Billing notification enabled.",
}

var NotificationSettingDeployment = schema.BoolAttribute{
	Computed:    true,
	Description: "Deployment notification enabled.",
}

var NotificationSettingEmailEnabled = schema.BoolAttribute{
	Computed:    true,
	Description: "Email notifications enabled.",
}

var NotificationSettingEmails = schema.ListAttribute{
	ElementType: types.StringType,
	Computed:    true,
	Description: "List of email addresses for notifications.",
}

var NotificationSettingInappEnabled = schema.BoolAttribute{
	Computed:    true,
	Description: "In-app notifications enabled.",
}

var NotificationSettingInvites = schema.BoolAttribute{
	Computed:    true,
	Description: "Invites notification enabled.",
}

var NotificationSettingPayment = schema.BoolAttribute{
	Computed:    true,
	Description: "Payment notification enabled.",
}

var NotificationSettingWebhookEnabled = schema.BoolAttribute{
	Computed:    true,
	Description: "Webhook notifications enabled.",
}

var NotificationSettingWebhooks = schema.ListAttribute{
	ElementType: types.MapType{ElemType: types.StringType},
	Computed:    true,
	Description: "List of webhook configurations.",
}
