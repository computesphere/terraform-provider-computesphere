package provider

import (
	"context"

	shared "github.com/computesphere/terraform-provider-computesphere/internal/provider/types/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
)

func Schema(ctx context.Context) schema.Schema {
	return schema.Schema{
		Description: "Provides details for a single alert.",
		Attributes: map[string]schema.Attribute{
			"id":                schema.StringAttribute{Required: true, Description: shared.AlertID.Description},
			"project_id":        schema.StringAttribute{Computed: true, Description: shared.ProjectID.Description},
			"environment_id":    schema.StringAttribute{Computed: true, Description: shared.EnvironmentID.Description},
			"alert_type":        schema.StringAttribute{Computed: true, Description: shared.AlertType.Description},
			"severity":          schema.StringAttribute{Computed: true, Description: shared.Severity.Description},
			"threshold":         schema.Int64Attribute{Computed: true, Description: shared.Threshold.Description},
			"evaluation_period": schema.Int64Attribute{Computed: true, Description: shared.EvaluationPeriod.Description},
			"active":            schema.BoolAttribute{Computed: true, Description: shared.Active.Description},
		},
	}
}

func PluralSchema(ctx context.Context) schema.Schema {
	return schema.Schema{
		Description: "Provides a list of alerts.",
		Attributes: map[string]schema.Attribute{
			"project_id": schema.StringAttribute{Required: true, Description: "Project to list alerts for."},
			"alerts": schema.ListNestedAttribute{
				Computed: true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"id":                shared.AlertID,
						"project_id":        schema.StringAttribute{Computed: true, Description: shared.ProjectID.Description},
						"environment_id":    schema.StringAttribute{Computed: true, Description: shared.EnvironmentID.Description},
						"alert_type":        schema.StringAttribute{Computed: true, Description: shared.AlertType.Description},
						"severity":          schema.StringAttribute{Computed: true, Description: shared.Severity.Description},
						"threshold":         schema.Int64Attribute{Computed: true, Description: shared.Threshold.Description},
						"evaluation_period": schema.Int64Attribute{Computed: true, Description: shared.EvaluationPeriod.Description},
						"active":            schema.BoolAttribute{Computed: true, Description: shared.Active.Description},
					},
				},
			},
		},
	}
}
