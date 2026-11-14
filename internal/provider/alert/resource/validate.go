package provider

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var (
	_ resource.ResourceWithValidateConfig = &AlertResource{}
	_ resource.ResourceWithModifyPlan     = &AlertResource{}
)

var metricTypes = map[string]bool{"cpu": true, "memory": true, "disk": true}

// ValidateConfig checks threshold against the alert type: a metric rule
// needs a percent from 1 to 100, and a failing-streak rule takes 2 to 10.
func (r *AlertResource) ValidateConfig(ctx context.Context, req resource.ValidateConfigRequest, resp *resource.ValidateConfigResponse) {
	var c alertResourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &c)...)
	if resp.Diagnostics.HasError() || c.AlertType.IsUnknown() || c.AlertType.IsNull() || c.Threshold.IsUnknown() {
		return
	}
	t := c.AlertType.ValueString()
	switch {
	case metricTypes[t] && c.Threshold.IsNull():
		resp.Diagnostics.AddAttributeError(path.Root("threshold"), "Missing threshold",
			"A "+t+" alert needs threshold, a percent from 1 to 100.")
	case metricTypes[t] && (c.Threshold.ValueInt64() < 1 || c.Threshold.ValueInt64() > 100):
		resp.Diagnostics.AddAttributeError(path.Root("threshold"), "Invalid threshold", "threshold must be from 1 to 100.")
	case t == "job_failing_streak" && !c.Threshold.IsNull() && (c.Threshold.ValueInt64() < 2 || c.Threshold.ValueInt64() > 10):
		resp.Diagnostics.AddAttributeError(path.Root("threshold"), "Invalid threshold",
			"For job_failing_streak, threshold is the failed runs in a row, from 2 to 10 (default 3).")
	}
}

// ModifyPlan: when alert_type changes, a threshold or evaluation_period the
// configuration leaves out is no longer the old rule's value, so it's
// unknown until the API answers instead of carried over.
func (r *AlertResource) ModifyPlan(ctx context.Context, req resource.ModifyPlanRequest, resp *resource.ModifyPlanResponse) {
	if req.State.Raw.IsNull() || req.Plan.Raw.IsNull() {
		return
	}
	var planType, stateType types.String
	resp.Diagnostics.Append(req.Plan.GetAttribute(ctx, path.Root("alert_type"), &planType)...)
	resp.Diagnostics.Append(req.State.GetAttribute(ctx, path.Root("alert_type"), &stateType)...)
	if resp.Diagnostics.HasError() || planType.Equal(stateType) {
		return
	}
	for _, attr := range []string{"threshold", "evaluation_period", "effective_evaluation_period"} {
		var cfg types.Int64
		resp.Diagnostics.Append(req.Config.GetAttribute(ctx, path.Root(attr), &cfg)...)
		if cfg.IsNull() {
			resp.Diagnostics.Append(resp.Plan.SetAttribute(ctx, path.Root(attr), types.Int64Unknown())...)
		}
	}
}
