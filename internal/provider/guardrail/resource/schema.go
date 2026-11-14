package provider

import (
	"context"

	shared "github.com/computesphere/terraform-provider-computesphere/internal/provider/types/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
)

func Schema(ctx context.Context) schema.Schema {
	return schema.Schema{
		Description:        "Deprecated: guardrails are set in the console. Manages a guardrail resource.",
		DeprecationMessage: "Guardrails are no longer part of the public API: set them in the console. This resource stops working once the API refuses API tokens for guardrails, and will be removed in a future release.",
		Attributes: map[string]schema.Attribute{
			"id":                     shared.GuardrailID,
			"name":                   shared.GuardrailName,
			"description":            shared.GuardrailDescription,
			"effect":                 shared.GuardrailEffect,
			"message":                shared.GuardrailMessage,
			"scope":                  shared.GuardrailScope,
			"status":                 shared.GuardrailStatus,
			"type":                   shared.GuardrailType,
			"account_id":             shared.GuardrailAccountID,
			"created_by":             shared.GuardrailCreatedBy,
			"is_predefined_assigned": shared.GuardrailIsPredefinedAssigned,
		},
	}
}
