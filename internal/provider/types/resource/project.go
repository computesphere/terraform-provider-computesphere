package resource

import (
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
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
var ProjectCreatedAt = schema.StringAttribute{
	Computed:    true,
	Description: "Creation timestamp of the project.",
}
