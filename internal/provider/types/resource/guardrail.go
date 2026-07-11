package resource

import (
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var GuardrailID = schema.StringAttribute{
	Computed:    true,
	Description: "Unique identifier for the guardrail.",
}

var GuardrailName = schema.StringAttribute{
	Required:    true,
	Description: "Name of the guardrail.",
}

var GuardrailDescription = schema.StringAttribute{
	Optional:    true,
	Description: "Description of the guardrail.",
}

var GuardrailEffect = schema.StringAttribute{
	Optional:    true,
	Description: "Effect of the guardrail.",
}

var GuardrailMessage = schema.StringAttribute{
	Optional:    true,
	Description: "Message for the guardrail.",
}

var GuardrailRules = schema.ListAttribute{
	ElementType: types.MapType{ElemType: types.StringType},
	Optional:    true,
	Description: "List of rules for the guardrail.",
}

var GuardrailScope = schema.StringAttribute{
	Optional:    true,
	Description: "Scope of the guardrail.",
}

var GuardrailStatus = schema.BoolAttribute{
	Optional:    true,
	Description: "Status of the guardrail.",
}

var GuardrailType = schema.StringAttribute{
	Optional:    true,
	Description: "Type of the guardrail.",
}

var GuardrailAccountID = schema.StringAttribute{
	Computed:    true,
	Description: "Account ID associated with the guardrail.",
}

var GuardrailCreatedBy = schema.StringAttribute{
	Computed:    true,
	Description: "User who created the guardrail.",
}

var GuardrailIsPredefinedAssigned = schema.BoolAttribute{
	Computed:    true,
	Description: "Whether the guardrail is predefined and assigned.",
}
