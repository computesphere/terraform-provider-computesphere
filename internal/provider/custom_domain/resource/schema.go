package provider

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
)

func Schema(ctx context.Context) schema.Schema {
	rr := []planmodifier.String{stringplanmodifier.RequiresReplace()}
	return schema.Schema{
		Description: "Manages a custom domain bound to a deployment (ADR 0132). Modeled as an " +
			"imperative sub-resource — one instance per hostname, so it is for_each-friendly. " +
			"Both the deployment and the hostname are immutable; changing either forces replacement.",
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
				Required:      true,
				Description:   "Fully-qualified hostname to bind to the deployment.",
				PlanModifiers: rr,
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
