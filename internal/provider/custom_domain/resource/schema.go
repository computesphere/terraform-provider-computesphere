package provider

import (
	"context"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
)

func Schema(ctx context.Context) schema.Schema {
	rr := []planmodifier.String{stringplanmodifier.RequiresReplace()}
	return schema.Schema{
		Description: "Manages a custom domain on a deployment. Use one resource per hostname (for_each works well). " +
			"Changing the deployment or the hostname replaces the domain.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:      true,
				Description:   "Server-issued identifier for the custom domain.",
				PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"deployment_id": schema.StringAttribute{
				Required:      true,
				Description:   "Identifier of the deployment this custom domain is bound to.",
				PlanModifiers: rr,
			},
			"domain": schema.StringAttribute{
				Required:    true,
				Description: "Fully-qualified hostname to bind to the deployment.",
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplaceIf(replaceIfHostnameDiffers,
					"Changing the hostname replaces the domain; a change of letter case doesn't.",
					"Changing the hostname replaces the domain; a change of letter case doesn't.")},
			},
			"hostname": schema.StringAttribute{
				Computed:      true,
				Description:   "Canonical fully-qualified hostname as reported by the server.",
				PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"status": schema.StringAttribute{
				Computed:    true,
				Description: "Reconciliation state (pending_dns, validating, issuing, active, action_required).",
			},
			"verified": schema.BoolAttribute{
				Computed:    true,
				Description: "True once the custom domain is active (verified and serving).",
			},
		},
	}
}

// Hostnames are case-insensitive: the API may hold a different spelling than
// the configuration (after an import, say), and that alone mustn't replace the
// domain.
func replaceIfHostnameDiffers(_ context.Context, req planmodifier.StringRequest, resp *stringplanmodifier.RequiresReplaceIfFuncResponse) {
	resp.RequiresReplace = hostnameDiffers(req.StateValue.ValueString(), req.PlanValue.ValueString())
}

func hostnameDiffers(a, b string) bool { return !strings.EqualFold(a, b) }
