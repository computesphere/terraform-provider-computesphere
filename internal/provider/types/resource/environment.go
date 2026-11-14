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
	Description:         "Region of the environment. It can't change after create; a change is refused at plan time.",
	MarkdownDescription: "Region of the environment. It can't change after create; a change is refused at plan time.",
}

var EnvironmentProjectID = schema.StringAttribute{
	Required:            true,
	Description:         "Project ID associated with the environment. Changing it moves the environment to that project in the same account. While the environment runs services the change is refused at plan time: move it in the console or with the CLI, then set the new project here and on its services.",
	MarkdownDescription: "Project ID associated with the environment. Changing it moves the environment to that project in the same account. While the environment runs services the change is refused at plan time: move it in the console or with the CLI, then set the new project here and on its services.",
}

var EnvironmentCreatedAt = schema.StringAttribute{
	Computed:            true,
	Description:         "Creation timestamp of the environment.",
	MarkdownDescription: "Creation timestamp of the environment.",
}
