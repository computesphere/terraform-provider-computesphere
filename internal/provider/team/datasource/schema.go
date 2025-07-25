package provider

import (
	"context"

	shared "github.com/computesphere/terraform-provider-computesphere/internal/provider/types/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
)

func Schema(ctx context.Context) schema.Schema {
	return schema.Schema{
		Description: "Provides details for a single team.",
		Attributes: map[string]schema.Attribute{
			"id":          shared.TeamID,
			"name":        shared.TeamName,
			"description": shared.TeamDescription,
			"created_at":  shared.TeamCreatedAt,
			"updated_at":  shared.TeamUpdatedAt,
			"account_id":  shared.TeamAccountID,
		},
	}
}

func PluralSchema(ctx context.Context) schema.Schema {
	return schema.Schema{
		Description: "Provides a list of teams.",
		Attributes: map[string]schema.Attribute{
			"teams": schema.ListNestedAttribute{
				Computed: true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"id":          shared.TeamID,
						"name":        shared.TeamName,
						"description": shared.TeamDescription,
						"created_at":  shared.TeamCreatedAt,
						"updated_at":  shared.TeamUpdatedAt,
						"account_id":  shared.TeamAccountID,
					},
				},
			},
		},
	}
}
