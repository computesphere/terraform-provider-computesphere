package provider

import (
	"context"

	"github.com/computesphere/terraform-provider-computesphere/internal/provider/common/dataproduct"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var _ resource.ResourceWithModifyPlan = &DatabaseResource{}

const threeZoneWarning = "three_zone is no longer offered for new databases. Use none or standby."

// ModifyPlan refuses changes that would replace the database (an empty one),
// and warns at plan time when a new database asks for three zones. The
// API refuses it, so the apply would fail. Existing databases that already have
// three zones aren't warned. The deprecated plan ha isn't three zones; it maps
// to Standard 2 with a standby.
func (r *DatabaseResource) ModifyPlan(ctx context.Context, req resource.ModifyPlanRequest, resp *resource.ModifyPlanResponse) {
	dataproduct.RefuseChange(ctx, req, resp, "database", "name", "environment_id", "engine", "version")
	if !req.State.Raw.IsNull() || req.Plan.Raw.IsNull() {
		return
	}
	var ha types.String
	resp.Diagnostics.Append(req.Config.GetAttribute(ctx, path.Root("high_availability"), &ha)...)
	if resp.Diagnostics.HasError() {
		return
	}
	if threeZoneRequested(ha) {
		resp.Diagnostics.AddAttributeWarning(path.Root("high_availability"), threeZoneWarning, "")
	}
}

func threeZoneRequested(ha types.String) bool {
	return ha.ValueString() == "three_zone"
}
