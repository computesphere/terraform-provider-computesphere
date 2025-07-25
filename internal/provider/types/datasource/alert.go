package datasource

import (
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
)

var AlertID = schema.StringAttribute{
	Required:            true,
	Description:         "Unique identifier for the alert.",
	MarkdownDescription: "Unique identifier for the alert.",
}

var ProjectID = schema.StringAttribute{
	Computed:            true,
	Description:         "ID of the project this alert belongs to.",
	MarkdownDescription: "ID of the project this alert belongs to.",
}

var EnvironmentID = schema.StringAttribute{
	Computed:            true,
	Description:         "ID of the environment this alert belongs to.",
	MarkdownDescription: "ID of the environment this alert belongs to.",
}

var AlertType = schema.StringAttribute{
	Computed:            true,
	Description:         "Type of the alert.",
	MarkdownDescription: "Type of the alert.",
}

var Severity = schema.StringAttribute{
	Computed:            true,
	Description:         "Severity level of the alert.",
	MarkdownDescription: "Severity level of the alert.",
}

var Threshold = schema.Int64Attribute{
	Computed:            true,
	Description:         "Threshold value for the alert.",
	MarkdownDescription: "Threshold value for the alert.",
}

var EvaluationPeriod = schema.Int64Attribute{
	Computed:            true,
	Description:         "Evaluation period for the alert.",
	MarkdownDescription: "Evaluation period for the alert.",
}

var Active = schema.BoolAttribute{
	Computed:            true,
	Description:         "Whether the alert is active.",
	MarkdownDescription: "Whether the alert is active.",
}
