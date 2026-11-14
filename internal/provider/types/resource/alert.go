package resource

import (
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64planmodifier"
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
	Required: true,
	Description: "What the rule watches: `cpu`, `memory` or `disk` (a metric of the environment's services over " +
		"`threshold` percent for the evaluation period), or `job_run_failed`, `job_failing_streak` (`threshold` " +
		"failed runs in a row) or `job_run_missed` for the environment's scheduled jobs.",
}

var Severity = schema.StringAttribute{
	Required:            true,
	Description:         "Severity level of the alert.",
	MarkdownDescription: "Severity level of the alert.",
}

var Threshold = schema.Int64Attribute{
	Optional: true, Computed: true,
	PlanModifiers: []planmodifier.Int64{int64planmodifier.UseStateForUnknown()},
	Description: "Percent for `cpu`, `memory` and `disk` (1 to 100); failed runs in a row for " +
		"`job_failing_streak` (2 to 10, default 3). `job_run_failed` and `job_run_missed` don't use it. " +
		"When `alert_type` changes and this is left out, the new rule's own value is used.",
}

var EvaluationPeriod = schema.Int64Attribute{
	Optional: true, Computed: true,
	PlanModifiers: []planmodifier.Int64{int64planmodifier.UseStateForUnknown()},
	Description: "Seconds the metric must stay over the threshold. Rules evaluate in whole minutes, at least 120 " +
		"seconds (see `effective_evaluation_period`). The job alert types don't use it.",
}

var Active = schema.BoolAttribute{
	Optional:            true,
	Computed:            true,
	Description:         "Whether the alert is active.",
	MarkdownDescription: "Whether the alert is active.",
}
