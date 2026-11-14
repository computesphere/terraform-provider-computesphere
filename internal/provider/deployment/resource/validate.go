package provider

import (
	"context"

	"github.com/computesphere/terraform-provider-computesphere/internal/provider/spherelet"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
)

var _ resource.ResourceWithValidateConfig = &DeploymentResource{}

// ValidateConfig refuses instance_type together with the deprecated plan_id.
func (r *DeploymentResource) ValidateConfig(ctx context.Context, req resource.ValidateConfigRequest, resp *resource.ValidateConfigResponse) {
	var c deploymentResourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &c)...)
	if resp.Diagnostics.HasError() {
		return
	}
	if !c.InstanceType.IsNull() && !c.PlanID.IsNull() {
		resp.Diagnostics.AddAttributeError(path.Root(spherelet.FieldPlanID), "Conflicting attributes", spherelet.ConflictDetail)
	}
}
