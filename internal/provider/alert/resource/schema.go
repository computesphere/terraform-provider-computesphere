package provider

import (
	"context"

	shared "github.com/computesphere/terraform-provider-computesphere/internal/provider/types/resource"

	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
)

func Schema(ctx context.Context) schema.Schema {
	return schema.Schema{
		Description: "Manages an alert resource.",
		Attributes: map[string]schema.Attribute{
			"id":                shared.AlertID,
			"name":              schema.StringAttribute{Required: true, Description: "Name of the alert."},
			"project_id":        shared.ProjectID,
			"environment_id":    shared.EnvironmentID,
			"alert_type":        shared.AlertType,
			"severity":          shared.Severity,
			"threshold":         shared.Threshold,
			"evaluation_period": shared.EvaluationPeriod,
			"active":            shared.Active,
		},
	}
}
