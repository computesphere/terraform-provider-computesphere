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

var RegionOutboundIPs = schema.ListAttribute{
	ElementType: types.StringType,
	Computed:    true,
	Description: "The addresses services in this region connect out from, as CIDR ranges, for allowlists on databases, email providers and partner APIs. Every service in the region shares them. Null until they are published.",
}

var RegionBuildOutboundIPs = schema.ListAttribute{
	ElementType: types.StringType,
	Computed:    true,
	Description: "The addresses builds connect out from, as CIDR ranges, for example when a build fetches from a package registry behind an allowlist. Every build on ComputeSphere shares them, so every region lists the same ranges. Null until they are published.",
}
