package resource

import (
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
)

var PlanID = schema.StringAttribute{
	Required:            true,
	Description:         "Unique identifier for the plan.",
	MarkdownDescription: "Unique identifier for the plan.",
}
var PlanActive = schema.BoolAttribute{
	Computed:            true,
	Description:         "Whether the plan is active.",
	MarkdownDescription: "Whether the plan is active.",
}
var PlanCore = schema.Int64Attribute{
	Computed:            true,
	Description:         "Number of cores for the plan.",
	MarkdownDescription: "Number of cores for the plan.",
}
var PlanCountryCode = schema.StringAttribute{
	Computed:            true,
	Description:         "Country code for the plan.",
	MarkdownDescription: "Country code for the plan.",
}
var PlanCurrencyCode = schema.StringAttribute{
	Computed:            true,
	Description:         "Currency code for the plan.",
	MarkdownDescription: "Currency code for the plan.",
}
var PlanMemory = schema.Int64Attribute{
	Computed:            true,
	Description:         "Memory for the plan.",
	MarkdownDescription: "Memory for the plan.",
}
var PlanName = schema.StringAttribute{
	Computed:            true,
	Description:         "Name of the plan.",
	MarkdownDescription: "Name of the plan.",
}
var PlanPrice = schema.Float64Attribute{
	Computed:            true,
	Description:         "Price of the plan.",
	MarkdownDescription: "Price of the plan.",
}
var PlanSlug = schema.StringAttribute{
	Computed:            true,
	Description:         "Slug of the plan.",
	MarkdownDescription: "Slug of the plan.",
}
var PlanType = schema.StringAttribute{
	Computed:            true,
	Description:         "Type of the plan.",
	MarkdownDescription: "Type of the plan.",
}
