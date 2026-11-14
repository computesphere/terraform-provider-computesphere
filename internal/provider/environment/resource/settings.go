package provider

import (
	"context"
	"net/http"

	csv2 "github.com/computesphere/computesphere-go"
	cstypes "github.com/computesphere/terraform-provider-computesphere/internal/provider/types"
	"github.com/google/uuid"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/boolplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/listplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// settingsAttributes are the environment settings beyond name, region and
// project: who can deploy, whether the default URL serves, and who can reach
// the environment.
func settingsAttributes() map[string]schema.Attribute {
	keepStr := []planmodifier.String{stringplanmodifier.UseStateForUnknown()}
	keepBool := []planmodifier.Bool{boolplanmodifier.UseStateForUnknown()}
	return map[string]schema.Attribute{
		"deploy_policy": schema.StringAttribute{
			Optional: true, Computed: true, PlanModifiers: keepStr,
			Description: "`auto` (the default): builds deploy when they finish. `approval`: builds wait for a project " +
				"admin to approve them, and image deploys need the project admin role. Changing it needs the project admin role.",
		},
		"default_url_disabled": schema.BoolAttribute{
			Optional: true, Computed: true, PlanModifiers: keepBool,
			Description: "Turn the environment's default `computesphere.app` URL off. Needs a custom domain that is " +
				"verified and serving. Removing the last serving custom domain turns the default URL back on by itself.",
		},
		"network_policy_enabled": schema.BoolAttribute{
			Computed: true, PlanModifiers: keepBool,
			Description: "Whether the environment's private network policy is on: its services reach each other " +
				"over its private network, and services in other environments can't reach them.",
		},
		"ip_allowlist": schema.ListAttribute{
			Optional: true, Computed: true, ElementType: types.StringType,
			PlanModifiers: []planmodifier.List{listplanmodifier.UseStateForUnknown()},
			Description: "CIDR ranges allowed to reach the environment. Set `[]` to make it public (the default); " +
				"leaving it out keeps the current list.",
		},
		"sub_domain": schema.StringAttribute{
			Computed: true, PlanModifiers: keepStr, Description: "The environment's generated subdomain.",
		},
		"kind": schema.StringAttribute{
			Computed: true, PlanModifiers: keepStr, Description: "`standard`, `preview` or `fork`.",
		},
	}
}

// applySettings maps the settings of an API environment into the model.
func (m *environmentResourceModel) applySettings(ctx context.Context, e *csv2.Environment, diags *diag.Diagnostics) {
	m.DeployPolicy = types.StringValue(string(e.DeployPolicy))
	m.DefaultURLDisabled = types.BoolValue(e.DefaultUrlDisabled != nil && *e.DefaultUrlDisabled)
	m.NetworkPolicyEnabled = types.BoolValue(e.NetworkPolicy != nil && e.NetworkPolicy.Enabled)
	ips := []string{}
	if e.NetworkAccess != nil && e.NetworkAccess.Type != "public_access" && e.NetworkAccess.Type != "public" {
		ips = append(ips, e.NetworkAccess.WhitelistedIps...)
	}
	l, d := types.ListValueFrom(ctx, types.StringType, ips)
	diags.Append(d...)
	m.IPAllowlist = l
	m.SubDomain = types.StringPointerValue(e.SubDomain)
	m.Kind = types.StringValue(string(e.Kind))
}

// changed reports whether plan sets a value to send: set, and different from
// prior (or there is no prior yet).
func changed(plan, prior attr.Value, hasPrior bool) bool {
	if plan.IsNull() || plan.IsUnknown() {
		return false
	}
	return !hasPrior || !plan.Equal(prior)
}

// writeSettings sends the settings the plan sets and that differ from prior
// (all set ones on create, when hasPrior is false). name, when non-nil, goes
// in the same PATCH. It returns the environment as the API last answered it,
// or current when nothing was sent.
func (r *EnvironmentResource) writeSettings(ctx context.Context, eid uuid.UUID, plan, prior environmentResourceModel, hasPrior bool, name *string, current *csv2.Environment, diags *diag.Diagnostics) *csv2.Environment {
	body := csv2.UpdateEnvironmentRequest{Name: name}
	patch := name != nil
	if changed(plan.DeployPolicy, prior.DeployPolicy, hasPrior) {
		p := csv2.UpdateEnvironmentRequestDeployPolicy(plan.DeployPolicy.ValueString())
		body.DeployPolicy = &p
		patch = true
	}
	if changed(plan.DefaultURLDisabled, prior.DefaultURLDisabled, hasPrior) {
		v := plan.DefaultURLDisabled.ValueBool()
		body.DefaultUrlDisabled = &v
		patch = true
	}
	if patch {
		apiResp, err := r.client.UpdateEnvironmentWithResponse(ctx, eid, body)
		if err != nil {
			diags.AddError("Error updating environment", err.Error())
			return nil
		}
		if apiResp.StatusCode() != http.StatusOK || apiResp.JSON200 == nil {
			diags.AddError("Error updating environment", cstypes.ProblemSummary(apiResp.Body, apiResp.StatusCode()))
			return nil
		}
		current = apiResp.JSON200
	}
	if changed(plan.IPAllowlist, prior.IPAllowlist, hasPrior) {
		var ips []string
		diags.Append(plan.IPAllowlist.ElementsAs(ctx, &ips, false)...)
		req := csv2.UpdateNetworkAccessRequest{Type: "public_access"}
		if len(ips) > 0 {
			req.Type = "restricted_access"
			req.WhitelistedIps = &ips
		}
		apiResp, err := r.client.UpdateEnvironmentNetworkingWithResponse(ctx, eid, req)
		if err != nil {
			diags.AddError("Error updating environment network access", err.Error())
			return nil
		}
		if apiResp.StatusCode() != http.StatusOK {
			diags.AddError("Error updating environment network access", cstypes.ProblemSummary(apiResp.Body, apiResp.StatusCode()))
			return nil
		}
	} else {
		return current
	}
	get, err := r.client.GetEnvironmentWithResponse(ctx, eid)
	if err != nil {
		diags.AddError("Error reading environment", err.Error())
		return nil
	}
	if get.StatusCode() != http.StatusOK || get.JSON200 == nil {
		diags.AddError("Error reading environment", cstypes.ProblemSummary(get.Body, get.StatusCode()))
		return nil
	}
	return get.JSON200
}
