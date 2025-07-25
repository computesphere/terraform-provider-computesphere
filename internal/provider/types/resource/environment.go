package resource

import (
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
)

var EnvID = schema.StringAttribute{
	Computed:            true,
	Description:         "Unique identifier for the environment.",
	MarkdownDescription: "Unique identifier for the environment.",
}

var EnvironmentName = schema.StringAttribute{
	Required:            true,
	Description:         "Name of the environment.",
	MarkdownDescription: "Name of the environment.",
}

var EnvironmentRegion = schema.StringAttribute{
	Required:            true,
	Description:         "Region of the environment.",
	MarkdownDescription: "Region of the environment.",
}

var EnvironmentProjectID = schema.StringAttribute{
	Required:            true,
	Description:         "Project ID associated with the environment.",
	MarkdownDescription: "Project ID associated with the environment.",
}

var EnvironmentCreatedAt = schema.StringAttribute{
	Computed:            true,
	Description:         "Creation timestamp of the environment.",
	MarkdownDescription: "Creation timestamp of the environment.",
}
