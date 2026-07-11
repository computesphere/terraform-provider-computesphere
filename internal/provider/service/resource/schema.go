package provider

import (
	"context"

	shared "github.com/computesphere/terraform-provider-computesphere/internal/provider/types/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
)

func Schema(ctx context.Context) schema.Schema {
	return schema.Schema{
		Description: "Manages a service resource.",
		Attributes: map[string]schema.Attribute{
			"id":             shared.ServiceID,
			"name":           shared.ServiceName,
			"project_id":     shared.ServiceProjectID,
			"type":           shared.ServiceType,
			"plan_id":        shared.ServicePlanID,
			"active":         shared.ServiceActive,
			"created_at":     shared.ServiceCreatedAt,
			"last_opened_at": shared.ServiceLastOpenedAt,
		},
	}
}
