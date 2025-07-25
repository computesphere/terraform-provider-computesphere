package datasource

import (
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
)

var ProjID = schema.StringAttribute{
	Required:    true,
	Description: "Unique identifier for the project.",
}
var ProjectName = schema.StringAttribute{
	Computed:    true,
	Description: "Name of the project.",
}
var ProjectDescription = schema.StringAttribute{
	Computed:    true,
	Description: "Description of the project.",
}
var ProjectPlanName = schema.StringAttribute{
	Computed:    true,
	Description: "Name of the plan for the project.",
}
var ProjectPlanValue = schema.Int64Attribute{
	Computed:    true,
	Description: "Plan value for the project.",
}
var ProjectPlanID = schema.StringAttribute{
	Computed:    true,
	Description: "Plan ID for the project.",
}
var ProjectCreatedAt = schema.StringAttribute{
	Computed:    true,
	Description: "Creation timestamp of the project.",
}
