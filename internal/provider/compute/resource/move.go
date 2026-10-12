package provider

import (
	"context"
	"fmt"
	"strings"

	deploymentresource "github.com/computesphere/terraform-provider-computesphere/internal/provider/deployment/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// deploymentState mirrors the computesphere_deployment schema, the source of
// a `moved {}` block.
type deploymentState struct {
	ID            types.String `tfsdk:"id"`
	Name          types.String `tfsdk:"name"`
	Status        types.String `tfsdk:"status"`
	ServiceID     types.String `tfsdk:"service_id"`
	EnvironmentID types.String `tfsdk:"environment_id"`
	ProjectID     types.String `tfsdk:"project_id"`
	Type          types.String `tfsdk:"type"`
	PlanID        types.String `tfsdk:"plan_id"`
	Image         types.String `tfsdk:"image"`
	ImageType     types.String `tfsdk:"image_type"`
	ImageProvider types.String `tfsdk:"image_provider"`
	ImageUsername types.String `tfsdk:"image_username"`
	ImagePassword types.String `tfsdk:"image_password"`
	ImageURL      types.String `tfsdk:"image_url"`
	BuildArgs     types.Map    `tfsdk:"build_args"`
	Port          types.Int64  `tfsdk:"port"`
	SphereCount   types.Int64  `tfsdk:"sphere_count"`
	EnvVars       types.Map    `tfsdk:"env_vars"`
	SecretVars    types.Map    `tfsdk:"secret_vars"`
	WaitForReady  types.Bool   `tfsdk:"wait_for_ready"`
}

// MoveState lets a `moved {}` block move a computesphere_deployment of this
// kind's type to this resource without touching the running deployment: the
// id and every field both schemas share carry over.
func (r *computeResource) MoveState(ctx context.Context) []resource.StateMover {
	src := deploymentresource.Schema(ctx)
	return []resource.StateMover{{
		SourceSchema: &src,
		StateMover: func(ctx context.Context, req resource.MoveStateRequest, resp *resource.MoveStateResponse) {
			if req.SourceTypeName != "computesphere_deployment" || req.SourceState == nil {
				return
			}
			var d deploymentState
			resp.Diagnostics.Append(req.SourceState.Get(ctx, &d)...)
			if resp.Diagnostics.HasError() {
				return
			}
			target, ok := moveFromDeployment(ctx, r.kind, d, resp)
			if !ok {
				return
			}
			obj, diags := toObject(ctx, r.kind, target)
			resp.Diagnostics.Append(diags...)
			if resp.Diagnostics.HasError() {
				return
			}
			resp.Diagnostics.Append(resp.TargetState.Set(ctx, obj)...)
		},
	}}
}

// kindFor returns the kind whose type a deployment has, if any.
func kindFor(apiType string) (kind, bool) {
	for _, k := range kinds {
		if k.matches(apiType) {
			return k, true
		}
	}
	return kind{}, false
}

func moveFromDeployment(ctx context.Context, k kind, d deploymentState, resp *resource.MoveStateResponse) (model, bool) {
	if !k.matches(d.Type.ValueString()) {
		hint := "It can only stay a computesphere_deployment."
		if other, ok := kindFor(d.Type.ValueString()); ok {
			hint = fmt.Sprintf("Move it to %s instead.", other.typeName)
		}
		resp.Diagnostics.AddError("Can't move this deployment here",
			fmt.Sprintf("Deployment %s is a %s, and %s manages %s deployments. %s",
				d.ID.ValueString(), d.Type.ValueString(), k.typeName, k.apiType, hint))
		return model{}, false
	}

	m, diags := fromObject(ctx, types.ObjectNull(nil))
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return model{}, false
	}
	m.ID, m.Name, m.Status = d.ID, d.Name, d.Status
	m.ServiceID, m.EnvironmentID, m.ProjectID = d.ServiceID, d.EnvironmentID, d.ProjectID
	m.EnvVars, m.SecretVars, m.WaitForReady = d.EnvVars, d.SecretVars, d.WaitForReady
	if m.WaitForReady.IsNull() {
		m.WaitForReady = types.BoolValue(true)
	}

	dropped := []string{}
	carry := func(attr string, set bool, apply func()) {
		if !set {
			return
		}
		if k.has(attr) {
			apply()
			return
		}
		dropped = append(dropped, attr)
	}
	carry("image", !d.Image.IsNull(), func() { m.Image = d.Image })
	carry("image_type", !d.ImageType.IsNull(), func() { m.ImageType = d.ImageType })
	carry("image_provider", !d.ImageProvider.IsNull(), func() { m.ImageProvider = d.ImageProvider })
	carry("image_username", !d.ImageUsername.IsNull(), func() { m.ImageUsername = d.ImageUsername })
	carry("image_password", !d.ImagePassword.IsNull(), func() { m.ImagePassword = d.ImagePassword })
	carry("image_url", !d.ImageURL.IsNull(), func() { m.ImageURL = d.ImageURL })
	carry("build_args", !d.BuildArgs.IsNull(), func() { m.BuildArgs = d.BuildArgs })
	carry("port", !d.Port.IsNull(), func() { m.Port = d.Port })
	carry("sphere_count", !d.SphereCount.IsNull(), func() { m.SphereCount = d.SphereCount })
	if !d.PlanID.IsNull() && d.PlanID.ValueString() != "" {
		dropped = append(dropped, "plan_id")
	}
	if len(dropped) > 0 {
		detail := fmt.Sprintf("%s has no %s. The deployment keeps running as it is; leave them out of the "+
			"configuration.", k.typeName, strings.Join(dropped, ", "))
		if !d.PlanID.IsNull() && d.PlanID.ValueString() != "" {
			detail += " The plan always comes from the service."
		}
		resp.Diagnostics.AddWarning("Some deployment settings don't carry over", detail)
	}
	return m, true
}
