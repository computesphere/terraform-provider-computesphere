package datasource

import (
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var GuardrailID = schema.StringAttribute{
	Required:            true,
	Description:         "Unique identifier for the guardrail.",
	MarkdownDescription: "Unique identifier for the guardrail.",
}

var GuardrailName = schema.StringAttribute{
	Computed:            true,
	Description:         "Name of the guardrail.",
	MarkdownDescription: "Name of the guardrail.",
}

var GuardrailDescription = schema.StringAttribute{
	Computed:            true,
	Description:         "Description of the guardrail.",
	MarkdownDescription: "Description of the guardrail.",
}

var GuardrailEffect = schema.StringAttribute{
	Computed:            true,
	Description:         "Effect of the guardrail.",
	MarkdownDescription: "Effect of the guardrail.",
}

var GuardrailMessage = schema.StringAttribute{
	Computed:            true,
	Description:         "Message for the guardrail.",
	MarkdownDescription: "Message for the guardrail.",
}

var GuardrailRules = schema.ListAttribute{
	ElementType:         types.MapType{ElemType: types.StringType},
	Computed:            true,
	Description:         "List of rules for the guardrail.",
	MarkdownDescription: "List of rules for the guardrail.",
}

var GuardrailScope = schema.StringAttribute{
	Computed:            true,
	Description:         "Scope of the guardrail.",
	MarkdownDescription: "Scope of the guardrail.",
}

var GuardrailStatus = schema.BoolAttribute{
	Computed:            true,
	Description:         "Status of the guardrail.",
	MarkdownDescription: "Status of the guardrail.",
}

var GuardrailType = schema.StringAttribute{
	Computed:            true,
	Description:         "Type of the guardrail.",
	MarkdownDescription: "Type of the guardrail.",
}

var GuardrailAccountID = schema.StringAttribute{
	Computed:            true,
	Description:         "Account ID associated with the guardrail.",
	MarkdownDescription: "Account ID associated with the guardrail.",
}

var GuardrailCreatedBy = schema.StringAttribute{
	Computed:            true,
	Description:         "User who created the guardrail.",
	MarkdownDescription: "User who created the guardrail.",
}

var GuardrailIsPredefinedAssigned = schema.BoolAttribute{
	Computed:            true,
	Description:         "Whether the guardrail is predefined and assigned.",
	MarkdownDescription: "Whether the guardrail is predefined and assigned.",
}
