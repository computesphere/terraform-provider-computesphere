package provider

import (
	"context"

	shared "github.com/computesphere/terraform-provider-computesphere/internal/provider/types/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
)

func Schema(ctx context.Context) schema.Schema {
	return schema.Schema{
		Description: "Manages a team resource.",
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
