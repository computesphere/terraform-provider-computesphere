package provider

import (
	"context"

	shared "github.com/computesphere/terraform-provider-computesphere/internal/provider/types/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
)

func Schema(ctx context.Context) schema.Schema {
	return schema.Schema{
		Description: "Provides details for a single plan.",
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

func PluralSchema(ctx context.Context) schema.Schema {
	return schema.Schema{
		Description: "Provides a list of plans.",
		Attributes: map[string]schema.Attribute{
			"plans": schema.ListNestedAttribute{
				Computed: true,
				NestedObject: schema.NestedAttributeObject{
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
				},
			},
		},
	}
}
