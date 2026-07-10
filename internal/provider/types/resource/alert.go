package resource

import (
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
)

var AlertID = schema.StringAttribute{
	Computed:            true,
	Description:         "Unique identifier for the alert.",
	MarkdownDescription: "Unique identifier for the alert.",
	PlanModifiers: []planmodifier.String{
		stringplanmodifier.UseStateForUnknown(),
	},
}

var ProjectID = schema.StringAttribute{
	Required:            true,
	Description:         "ID of the project this alert belongs to.",
	MarkdownDescription: "ID of the project this alert belongs to.",
	PlanModifiers: []planmodifier.String{
		stringplanmodifier.RequiresReplace(),
	},
}

var EnvironmentID = schema.StringAttribute{
	Required:            true,
	Description:         "ID of the environment this alert belongs to.",
	MarkdownDescription: "ID of the environment this alert belongs to.",
}

var AlertType = schema.StringAttribute{
	Required:            true,
	Description:         "Type of the alert.",
	MarkdownDescription: "Type of the alert.",
}

var Severity = schema.StringAttribute{
	Required:            true,
	Description:         "Severity level of the alert.",
	MarkdownDescription: "Severity level of the alert.",
}

var Threshold = schema.Int64Attribute{
	Required:            true,
	Description:         "Threshold value for the alert.",
	MarkdownDescription: "Threshold value for the alert.",
}

var EvaluationPeriod = schema.Int64Attribute{
	Required:            true,
	Description:         "Evaluation period for the alert.",
	MarkdownDescription: "Evaluation period for the alert.",
}

var Active = schema.BoolAttribute{
	Optional:            true,
	Computed:            true,
	Description:         "Whether the alert is active.",
	MarkdownDescription: "Whether the alert is active.",
}
