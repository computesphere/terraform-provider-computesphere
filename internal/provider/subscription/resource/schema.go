package provider

import (
	"context"

	shared "github.com/computesphere/terraform-provider-computesphere/internal/provider/types/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
)

func Schema(ctx context.Context) schema.Schema {
	return schema.Schema{
		Description: "Manages a subscription resource.",
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
