package provider

import (
	"context"

	shared "github.com/computesphere/terraform-provider-computesphere/internal/provider/types/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
)

func Schema(ctx context.Context) schema.Schema {
	return schema.Schema{
		Description: "Manages a project resource.",
		Attributes: map[string]schema.Attribute{
			"id":          shared.ProjID,
			"name":        shared.ProjectName,
			"description": shared.ProjectDescription,
			"plan_name":   shared.ProjectPlanName,
			"plan_value":  shared.ProjectPlanValue,
			"plan_id":     shared.ProjectPlanID,
			"created_at":  shared.ProjectCreatedAt,
		},
	}
}
