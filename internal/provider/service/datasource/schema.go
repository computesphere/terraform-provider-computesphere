package provider

import (
	"context"

	shared "github.com/computesphere/terraform-provider-computesphere/internal/provider/types/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
)

func Schema(ctx context.Context) schema.Schema {
	return schema.Schema{
		Description: "Provides details for a single service.",
		Attributes: map[string]schema.Attribute{
			"id":             shared.ServiceID,
			"name":           shared.ServiceName,
			"project_id":     shared.ServiceProjectID,
			"type":           shared.ServiceType,
			"active":         shared.ServiceActive,
			"created_at":     shared.ServiceCreatedAt,
			"last_opened_at": shared.ServiceLastOpenedAt,
		},
	}
}

func PluralSchema(ctx context.Context) schema.Schema {
	return schema.Schema{
		Description: "Provides a list of services.",
		Attributes: map[string]schema.Attribute{
			"services": schema.ListNestedAttribute{
				Computed: true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"id":             shared.ServiceID,
						"name":           shared.ServiceName,
						"project_id":     shared.ServiceProjectID,
						"type":           shared.ServiceType,
						"active":         shared.ServiceActive,
						"created_at":     shared.ServiceCreatedAt,
						"last_opened_at": shared.ServiceLastOpenedAt,
					},
				},
			},
		},
	}
}
