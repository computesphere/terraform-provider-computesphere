package provider

import (
	"context"

	shared "github.com/computesphere/terraform-provider-computesphere/internal/provider/types/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
)

func Schema(ctx context.Context) schema.Schema {
	return schema.Schema{
		Description: "Provides details for a single project.",
		Attributes: map[string]schema.Attribute{
			"id":          schema.StringAttribute{Required: true, Description: shared.ProjID.Description},
			"name":        schema.StringAttribute{Computed: true, Description: shared.ProjectName.Description},
			"description": schema.StringAttribute{Computed: true, Description: shared.ProjectDescription.Description},
			"plan_name":   schema.StringAttribute{Computed: true, Description: shared.ProjectPlanName.Description},
			"plan_value":  schema.Int64Attribute{Computed: true, Description: shared.ProjectPlanValue.Description},
			"plan_id":     schema.StringAttribute{Computed: true, Description: shared.ProjectPlanID.Description},
			"created_at":  schema.StringAttribute{Computed: true, Description: shared.ProjectCreatedAt.Description},
		},
	}
}

func PluralSchema(ctx context.Context) schema.Schema {
	return schema.Schema{
		Description: "Provides a list of projects.",
		Attributes: map[string]schema.Attribute{
			"projects": schema.ListNestedAttribute{
				Computed: true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"id":          schema.StringAttribute{Computed: true, Description: shared.ProjID.Description},
						"name":        schema.StringAttribute{Computed: true, Description: shared.ProjectName.Description},
						"description": schema.StringAttribute{Computed: true, Description: shared.ProjectDescription.Description},
						"plan_id":     schema.StringAttribute{Computed: true, Description: shared.ProjectPlanID.Description},
						"created_at":  schema.StringAttribute{Computed: true, Description: shared.ProjectCreatedAt.Description},
					},
				},
			},
		},
	}
}
