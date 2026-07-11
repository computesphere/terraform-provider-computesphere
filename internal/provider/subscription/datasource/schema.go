package provider

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
)

func Schema(ctx context.Context) schema.Schema {
	return schema.Schema{
		Description: "Provides details for a single subscription.",
		Attributes: map[string]schema.Attribute{
			"id":            schema.StringAttribute{Required: true, Description: "Unique identifier for the subscription."},
			"name":          schema.StringAttribute{Computed: true, Description: "Name of the subscription."},
			"active":        schema.BoolAttribute{Computed: true, Description: "Whether the subscription is active."},
			"country_code":  schema.StringAttribute{Computed: true, Description: "Country code for the subscription."},
			"currency_code": schema.StringAttribute{Computed: true, Description: "Currency code for the subscription."},
			"price":         schema.Float64Attribute{Computed: true, Description: "Price of the subscription."},
		},
	}
}

func PluralSchema(ctx context.Context) schema.Schema {
	return schema.Schema{
		Description: "Provides a list of subscriptions.",
		Attributes: map[string]schema.Attribute{
			"subscriptions": schema.ListNestedAttribute{
				Computed: true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"id":            schema.StringAttribute{Computed: true, Description: "Unique identifier for the subscription."},
						"name":          schema.StringAttribute{Computed: true, Description: "Name of the subscription."},
						"active":        schema.BoolAttribute{Computed: true, Description: "Whether the subscription is active."},
						"country_code":  schema.StringAttribute{Computed: true, Description: "Country code for the subscription."},
						"currency_code": schema.StringAttribute{Computed: true, Description: "Currency code for the subscription."},
						"price":         schema.Float64Attribute{Computed: true, Description: "Price of the subscription."},
					},
				},
			},
		},
	}
}
