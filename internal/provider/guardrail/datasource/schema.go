package provider

import (
	"context"

	shared "github.com/computesphere/terraform-provider-computesphere/internal/provider/types/resource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
)

func Schema(ctx context.Context) schema.Schema {
	return schema.Schema{
		Description: "Provides details for a single guardrail.",
		Attributes: map[string]schema.Attribute{
			"id":                     schema.StringAttribute{Required: true, Description: shared.GuardrailID.Description},
			"name":                   schema.StringAttribute{Computed: true, Description: shared.GuardrailName.Description},
			"description":            schema.StringAttribute{Computed: true, Description: shared.GuardrailDescription.Description},
			"effect":                 schema.StringAttribute{Computed: true, Description: shared.GuardrailEffect.Description},
			"message":                schema.StringAttribute{Computed: true, Description: shared.GuardrailMessage.Description},
			"scope":                  schema.StringAttribute{Computed: true, Description: shared.GuardrailScope.Description},
			"status":                 schema.BoolAttribute{Computed: true, Description: shared.GuardrailStatus.Description},
			"type":                   schema.StringAttribute{Computed: true, Description: shared.GuardrailType.Description},
			"account_id":             schema.StringAttribute{Computed: true, Description: shared.GuardrailAccountID.Description},
			"created_by":             schema.StringAttribute{Computed: true, Description: shared.GuardrailCreatedBy.Description},
			"is_predefined_assigned": schema.BoolAttribute{Computed: true, Description: shared.GuardrailIsPredefinedAssigned.Description},
		},
	}
}
