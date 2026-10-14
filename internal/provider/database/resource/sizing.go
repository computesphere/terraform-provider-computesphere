package provider

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// sizingAttrs are the attributes the API derives from one another: changing the
// instance type or high availability moves the deprecated plan, and changing
// the plan moves the instance type and high availability.
var sizingAttrs = []string{"instance_type", "high_availability", "plan"}

// followsSizing keeps an unset sizing attribute at its prior value, like
// UseStateForUnknown, unless the configuration changes one of the other sizing
// attributes. Then the value is unknown until the API answers.
type followsSizing struct{ self string }

func (m followsSizing) Description(context.Context) string {
	return "Keeps the prior value unless another sizing attribute changes."
}

func (m followsSizing) MarkdownDescription(ctx context.Context) string { return m.Description(ctx) }

func (m followsSizing) PlanModifyString(ctx context.Context, req planmodifier.StringRequest, resp *planmodifier.StringResponse) {
	if req.StateValue.IsNull() || !req.PlanValue.IsUnknown() || req.ConfigValue.IsUnknown() || !req.ConfigValue.IsNull() {
		return
	}
	for _, other := range sizingAttrs {
		if other == m.self {
			continue
		}
		var cfg, prior types.String
		resp.Diagnostics.Append(req.Config.GetAttribute(ctx, path.Root(other), &cfg)...)
		resp.Diagnostics.Append(req.State.GetAttribute(ctx, path.Root(other), &prior)...)
		if resp.Diagnostics.HasError() {
			return
		}
		if cfg.IsUnknown() || (!cfg.IsNull() && !cfg.Equal(prior)) {
			return // stays unknown: the API sets it
		}
	}
	resp.PlanValue = req.StateValue
}

func sizing(self string) []planmodifier.String {
	return []planmodifier.String{followsSizing{self: self}}
}
