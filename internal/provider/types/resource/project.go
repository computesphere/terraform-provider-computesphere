package resource

import (
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64default"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
)

var ProjID = schema.StringAttribute{
	Computed:    true,
	Description: "Unique identifier for the project.",
}
var ProjectName = schema.StringAttribute{
	Required:    true,
	Description: "Name of the project.",
}
var ProjectDescription = schema.StringAttribute{
	Optional:    true,
	Computed:    true,
	Description: "Description of the project.",
	Default:     stringdefault.StaticString(""),
}
var ProjectPlanName = schema.StringAttribute{
	Optional:    true,
	Computed:    true,
	Description: "Name of the plan for the project.",
	Default:     stringdefault.StaticString("FLX"),
	PlanModifiers: []planmodifier.String{
		stringplanmodifier.RequiresReplace(),
	},
}
var ProjectPlanValue = schema.Int64Attribute{
	Optional:    true,
	Computed:    true,
	Description: "Plan value for the project.",
	Default:     int64default.StaticInt64(2),
}
var ProjectPlanID = schema.StringAttribute{
	Computed:    true,
	Description: "Plan ID for the project.",
}
var ProjectCreatedAt = schema.StringAttribute{
	Computed:    true,
	Description: "Creation timestamp of the project.",
}
