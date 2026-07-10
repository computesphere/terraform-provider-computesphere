package resource

import (
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
)

var ServiceID = schema.StringAttribute{
	Computed:    true,
	Description: "Unique identifier for the service.",
}
var ServiceName = schema.StringAttribute{
	Required:    true,
	Description: "Name of the service.",
	PlanModifiers: []planmodifier.String{
		stringplanmodifier.RequiresReplace(),
	},
}
var ServiceProjectID = schema.StringAttribute{
	Required:    true,
	Description: "Project the service belongs to.",
	PlanModifiers: []planmodifier.String{
		stringplanmodifier.RequiresReplace(),
	},
}
var ServiceType = schema.StringAttribute{
	Required:    true,
	Description: "Type of the service (e.g. web-service, cron-job, background-worker).",
	PlanModifiers: []planmodifier.String{
		stringplanmodifier.RequiresReplace(),
	},
}
var ServicePlanID = schema.StringAttribute{
	Optional:    true,
	Computed:    true,
	Description: "Plan (spherelet shape) assigned to the service. Updatable in place.",
}
var ServiceActive = schema.BoolAttribute{
	Computed:    true,
	Description: "Whether the service is active.",
}
var ServiceCreatedAt = schema.StringAttribute{
	Computed:    true,
	Description: "Creation timestamp of the service.",
}
var ServiceLastOpenedAt = schema.StringAttribute{
	Computed:    true,
	Description: "Last opened timestamp of the service.",
}
