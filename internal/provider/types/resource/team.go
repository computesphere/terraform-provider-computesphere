package resource

import (
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
)

var TeamID = schema.StringAttribute{
	Computed:    true,
	Description: "Unique identifier for the team.",
}
var TeamName = schema.StringAttribute{
	Required:    true,
	Description: "Name of the team.",
}
var TeamDescription = schema.StringAttribute{
	Optional:    true,
	Description: "Description of the team.",
}
var TeamCreatedAt = schema.StringAttribute{
	Computed:    true,
	Description: "Creation timestamp of the team.",
}
var TeamUpdatedAt = schema.StringAttribute{
	Computed:    true,
	Description: "Last updated timestamp of the team.",
}
var TeamAccountID = schema.StringAttribute{
	Optional:    true,
	Description: "The ComputeSphere account ID. Defaults to the provider account_id if not set.",
}
