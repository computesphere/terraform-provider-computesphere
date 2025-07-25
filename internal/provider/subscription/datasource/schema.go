package provider

import (
	"context"

	shared "github.com/computesphere/terraform-provider-computesphere/internal/provider/types/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
)

func Schema(ctx context.Context) schema.Schema {
	return schema.Schema{
		Description: "Provides details for a single subscription.",
		Attributes: map[string]schema.Attribute{
			"id":         shared.SubscriptionID,
			"name":       shared.SubscriptionName,
			"active":     shared.SubscriptionActive,
			"created_at": shared.SubscriptionCreatedAt,
			"type":       shared.SubscriptionType,
			"plan":       shared.SubscriptionPlan,
			"user_id":    shared.SubscriptionUserID,
		},
	}
}

func PluralSchema(ctx context.Context) schema.Schema {
	return schema.Schema{
		Description: "Provides a list of subscriptions.",
		Attributes: map[string]schema.Attribute{
			"subscriptions": schema.ListNestedAttribute{
				Computed: true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"id":         shared.SubscriptionID,
						"name":       shared.SubscriptionName,
						"active":     shared.SubscriptionActive,
						"created_at": shared.SubscriptionCreatedAt,
						"type":       shared.SubscriptionType,
						"plan":       shared.SubscriptionPlan,
						"user_id":    shared.SubscriptionUserID,
					},
				},
			},
		},
	}
}
