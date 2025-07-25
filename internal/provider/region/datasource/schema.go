package provider

import (
	"context"

	shared "github.com/computesphere/terraform-provider-computesphere/internal/provider/types/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
)

func Schema(ctx context.Context) schema.Schema {
	return schema.Schema{
		Description: "Provides details for a single region.",
		Attributes: map[string]schema.Attribute{
			"name": shared.RegionName,
		},
	}
}

func PluralSchema(ctx context.Context) schema.Schema {
	return schema.Schema{
		Description: "Provides a list of regions.",
		Attributes: map[string]schema.Attribute{
			"regions": shared.RegionsList,
		},
	}
}
