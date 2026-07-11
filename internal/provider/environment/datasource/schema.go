package provider

import (
	"context"

	shared "github.com/computesphere/terraform-provider-computesphere/internal/provider/types/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
)

func Schema(ctx context.Context) schema.Schema {
	return schema.Schema{
		Description: "Provides details for a single environment.",
		Attributes: map[string]schema.Attribute{
			"id":         schema.StringAttribute{Required: true, Description: shared.EnvID.Description},
			"name":       schema.StringAttribute{Computed: true, Description: shared.EnvironmentName.Description},
			"region":     schema.StringAttribute{Computed: true, Description: shared.EnvironmentRegion.Description},
			"project_id": schema.StringAttribute{Computed: true, Description: shared.EnvironmentProjectID.Description},
			"created_at": schema.StringAttribute{Computed: true, Description: shared.EnvironmentCreatedAt.Description},
		},
	}
}

func PluralSchema(ctx context.Context) schema.Schema {
	return schema.Schema{
		Description: "Provides a list of environments in a project.",
		Attributes: map[string]schema.Attribute{
			"project_id": schema.StringAttribute{Required: true, Description: "Project to list environments for."},
			"environments": schema.ListNestedAttribute{
				Computed: true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"id":         schema.StringAttribute{Computed: true, Description: shared.EnvID.Description},
						"name":       schema.StringAttribute{Computed: true, Description: shared.EnvironmentName.Description},
						"region":     schema.StringAttribute{Computed: true, Description: shared.EnvironmentRegion.Description},
						"project_id": schema.StringAttribute{Computed: true, Description: shared.EnvironmentProjectID.Description},
						"created_at": schema.StringAttribute{Computed: true, Description: shared.EnvironmentCreatedAt.Description},
					},
				},
			},
		},
	}
}
