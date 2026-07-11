package provider

import (
	"context"

	shared "github.com/computesphere/terraform-provider-computesphere/internal/provider/types/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
)

func Schema(ctx context.Context) schema.Schema {
	return schema.Schema{
		Description: "Manages an environment resource.",
		Attributes: map[string]schema.Attribute{
			"id":         shared.EnvID,
			"name":       shared.EnvironmentName,
			"region":     shared.EnvironmentRegion,
			"project_id": shared.EnvironmentProjectID,
			"created_at": shared.EnvironmentCreatedAt,
		},
	}
}
