package resource

import (
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
)

var ServiceID = schema.StringAttribute{
	Computed:    true,
	Description: "Unique identifier for the service.",
}
var ServiceName = schema.StringAttribute{
	Required:    true,
	Description: "Name of the service.",
}
var ServiceProjectID = schema.StringAttribute{
	Computed:    true,
	Description: "Project ID associated with the service.",
}
var ServiceType = schema.StringAttribute{
	Computed:    true,
	Description: "Type of the service.",
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
