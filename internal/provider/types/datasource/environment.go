package datasource

import (
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
)

var EnvID = schema.StringAttribute{
	Required:            true,
	Description:         "Unique identifier for the environment.",
	MarkdownDescription: "Unique identifier for the environment.",
}

var EnvironmentName = schema.StringAttribute{
	Computed:            true,
	Description:         "Name of the environment.",
	MarkdownDescription: "Name of the environment.",
}

var EnvironmentRegion = schema.StringAttribute{
	Computed:            true,
	Description:         "Region of the environment.",
	MarkdownDescription: "Region of the environment.",
}

var EnvironmentProjectID = schema.StringAttribute{
	Computed:            true,
	Description:         "Project ID associated with the environment.",
	MarkdownDescription: "Project ID associated with the environment.",
}

var EnvironmentCreatedAt = schema.StringAttribute{
	Computed:            true,
	Description:         "Creation timestamp of the environment.",
	MarkdownDescription: "Creation timestamp of the environment.",
}
