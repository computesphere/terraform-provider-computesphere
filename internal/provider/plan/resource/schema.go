package provider

import (
	"context"

	shared "github.com/computesphere/terraform-provider-computesphere/internal/provider/types/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
)

func Schema(ctx context.Context) schema.Schema {
	return schema.Schema{
		Description: "Manages a plan resource.",
		Attributes: map[string]schema.Attribute{
			"id":            shared.PlanID,
			"active":        shared.PlanActive,
			"core":          shared.PlanCore,
			"country_code":  shared.PlanCountryCode,
			"currency_code": shared.PlanCurrencyCode,
			"memory":        shared.PlanMemory,
			"name":          shared.PlanName,
			"price":         shared.PlanPrice,
			"slug":          shared.PlanSlug,
			"type":          shared.PlanType,
		},
	}
}
