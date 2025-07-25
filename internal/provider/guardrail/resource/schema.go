package provider

import (
	"context"

	shared "github.com/computesphere/terraform-provider-computesphere/internal/provider/types/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
)

func Schema(ctx context.Context) schema.Schema {
	return schema.Schema{
		Description: "Manages a guardrail resource.",
		Attributes: map[string]schema.Attribute{
			"id":                     shared.GuardrailID,
			"name":                   shared.GuardrailName,
			"description":            shared.GuardrailDescription,
			"effect":                 shared.GuardrailEffect,
			"message":                shared.GuardrailMessage,
			"rules":                  shared.GuardrailRules,
			"scope":                  shared.GuardrailScope,
			"status":                 shared.GuardrailStatus,
			"type":                   shared.GuardrailType,
			"account_id":             shared.GuardrailAccountID,
			"created_by":             shared.GuardrailCreatedBy,
			"is_predefined_assigned": shared.GuardrailIsPredefinedAssigned,
		},
	}
}
