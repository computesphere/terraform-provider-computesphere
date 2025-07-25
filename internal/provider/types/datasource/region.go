package datasource

import (
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var RegionName = schema.StringAttribute{
	Required:    true,
	Description: "Name of the region.",
}

var RegionsList = schema.ListAttribute{
	ElementType: types.StringType,
	Computed:    true,
	Description: "List of available regions.",
}
