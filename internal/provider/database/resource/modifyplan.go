package provider

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var _ resource.ResourceWithModifyPlan = &DatabaseResource{}

const threeZoneWarning = "three_zone is no longer offered for new databases. Use none or standby."

// ModifyPlan warns at plan time when a new database asks for three zones, by
// high_availability or by the deprecated plan ha. Existing databases that
// already have three zones aren't warned.
func (r *DatabaseResource) ModifyPlan(ctx context.Context, req resource.ModifyPlanRequest, resp *resource.ModifyPlanResponse) {
	if !req.State.Raw.IsNull() || req.Plan.Raw.IsNull() {
		return
	}
	var ha, plan types.String
	resp.Diagnostics.Append(req.Config.GetAttribute(ctx, path.Root("high_availability"), &ha)...)
	resp.Diagnostics.Append(req.Config.GetAttribute(ctx, path.Root("plan"), &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	if threeZoneRequested(ha, plan) {
		attr := path.Root("high_availability")
		if plan.ValueString() == "ha" {
			attr = path.Root("plan")
		}
		resp.Diagnostics.AddAttributeWarning(attr, threeZoneWarning, "")
	}
}

func threeZoneRequested(ha, plan types.String) bool {
	return ha.ValueString() == "three_zone" || plan.ValueString() == "ha"
}
