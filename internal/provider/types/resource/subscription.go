package resource

import (
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
)

var SubscriptionID = schema.StringAttribute{
	Computed:    true,
	Description: "Unique identifier for the subscription.",
}
var SubscriptionName = schema.StringAttribute{
	Required:    true,
	Description: "Name of the subscription.",
}
var SubscriptionActive = schema.BoolAttribute{
	Computed:    true,
	Description: "Whether the subscription is active.",
}
var SubscriptionCreatedAt = schema.StringAttribute{
	Computed:    true,
	Description: "Creation timestamp of the subscription.",
}
var SubscriptionType = schema.StringAttribute{
	Computed:    true,
	Description: "Type of the subscription.",
}
var SubscriptionPlan = schema.StringAttribute{
	Computed:    true,
	Description: "Plan associated with the subscription.",
}
var SubscriptionUserID = schema.StringAttribute{
	Computed:    true,
	Description: "User ID associated with the subscription.",
}
