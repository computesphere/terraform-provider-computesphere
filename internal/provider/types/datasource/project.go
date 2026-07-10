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
var ProjectCreatedAt = schema.StringAttribute{
	Computed:    true,
	Description: "Creation timestamp of the project.",
}
