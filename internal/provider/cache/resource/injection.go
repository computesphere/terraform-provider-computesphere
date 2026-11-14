package provider

import (
	"context"
	"net/http"
	"net/url"
	"regexp"

	cstypes "github.com/computesphere/terraform-provider-computesphere/internal/provider/types"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/objectplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
)

var injectionAttrTypes = map[string]attr.Type{
	"mode":          types.StringType,
	"variable_name": types.StringType,
	"service_ids":   types.SetType{ElemType: types.StringType},
}

// injectionAttribute is the cache's injection block: which services in its
// environment get its URL as a variable, and under which name.
func injectionAttribute() schema.SingleNestedAttribute {
	return schema.SingleNestedAttribute{
		Optional: true, Computed: true,
		PlanModifiers: []planmodifier.Object{objectplanmodifier.UseStateForUnknown()},
		Description: "Which services in the cache's environment get its URL (a secret `rediss://` URL with the " +
			"password) as a variable. A new cache gives it to every service as `REDIS_URL` unless the environment " +
			"already has that name. Changing it restarts the services it affects. Leave it out to keep it as it is.",
		Attributes: map[string]schema.Attribute{
			"mode": schema.StringAttribute{
				Optional: true, Computed: true,
				Description: "`all` (every app service in the environment), `selected` (the services in `service_ids`) or `none`.",
			},
			"variable_name": schema.StringAttribute{
				Optional: true, Computed: true,
				Description: "Variable holding the cache's URL. Defaults to `REDIS_URL`.",
			},
			"service_ids": schema.SetAttribute{
				Optional: true, Computed: true, ElementType: types.StringType,
				Description: "Services that get the variable when `mode` is `selected`; empty otherwise.",
			},
		},
	}
}

type apiInjection struct {
	Mode         string   `json:"mode"`
	VariableName string   `json:"variable_name"`
	ServiceIDs   []string `json:"service_ids"`
	Notice       string   `json:"notice"`
}

// readInjection reads the cache's injection as the injection block's value.
func (r *CacheResource) readInjection(ctx context.Context, id string, diags *diag.Diagnostics) types.Object {
	var in apiInjection
	status, raw, err := cstypes.DoJSON(ctx, r.client, http.MethodGet, "/caches/"+url.PathEscape(id)+"/injection", nil, nil, &in)
	if err != nil {
		diags.AddError("Error reading cache injection", err.Error())
		return types.ObjectNull(injectionAttrTypes)
	}
	if status != http.StatusOK {
		diags.AddError("Error reading cache injection", cstypes.ProblemSummary(raw, status))
		return types.ObjectNull(injectionAttrTypes)
	}
	if in.Notice != "" {
		diags.AddWarning("Cache URL not in the environment", in.Notice)
	}
	if in.ServiceIDs == nil {
		in.ServiceIDs = []string{}
	}
	ids, d := types.SetValueFrom(ctx, types.StringType, in.ServiceIDs)
	diags.Append(d...)
	obj, d := types.ObjectValue(injectionAttrTypes, map[string]attr.Value{
		"mode": types.StringValue(in.Mode), "variable_name": types.StringValue(in.VariableName), "service_ids": ids,
	})
	diags.Append(d...)
	return obj
}

// writeInjection sends the configured injection when it differs from prior
// (or there is no prior). Unknown fields aren't sent: the API keeps them.
func (r *CacheResource) writeInjection(ctx context.Context, id string, plan, prior types.Object, diags *diag.Diagnostics) {
	if plan.IsNull() || plan.IsUnknown() || (!prior.IsNull() && !prior.IsUnknown() && plan.Equal(prior)) {
		return
	}
	var m struct {
		Mode         types.String `tfsdk:"mode"`
		VariableName types.String `tfsdk:"variable_name"`
		ServiceIDs   types.Set    `tfsdk:"service_ids"`
	}
	diags.Append(plan.As(ctx, &m, basetypes.ObjectAsOptions{UnhandledUnknownAsEmpty: true})...)
	if diags.HasError() {
		return
	}
	body := map[string]any{}
	if !m.Mode.IsNull() && !m.Mode.IsUnknown() {
		body["mode"] = m.Mode.ValueString()
	} else {
		// The API needs mode on every write: keep the cache's current one.
		cur := r.readInjection(ctx, id, diags)
		if diags.HasError() {
			return
		}
		if v, ok := cur.Attributes()["mode"].(types.String); ok {
			body["mode"] = v.ValueString()
		}
	}
	if !m.VariableName.IsNull() && !m.VariableName.IsUnknown() {
		body["variable_name"] = m.VariableName.ValueString()
	}
	if !m.ServiceIDs.IsNull() && !m.ServiceIDs.IsUnknown() {
		var ids []string
		diags.Append(m.ServiceIDs.ElementsAs(ctx, &ids, false)...)
		body["service_ids"] = ids
	}
	if len(body) == 0 {
		return
	}
	status, raw, err := cstypes.DoJSON(ctx, r.client, http.MethodPut, "/caches/"+url.PathEscape(id)+"/injection", nil, body, nil)
	if err != nil {
		diags.AddError("Error changing cache injection", err.Error())
		return
	}
	if status != http.StatusOK {
		diags.AddError("Error changing cache injection", cstypes.ProblemSummary(raw, status))
	}
}

var variableName = regexp.MustCompile(`^[A-Z][A-Z0-9_]*$`)

// validateInjection checks the injection block's mode and variable name.
func validateInjection(obj types.Object, diags *diag.Diagnostics) {
	if obj.IsNull() || obj.IsUnknown() {
		return
	}
	attrs := obj.Attributes()
	if v, ok := attrs["mode"].(types.String); ok && !v.IsNull() && !v.IsUnknown() {
		if m := v.ValueString(); m != "all" && m != "selected" && m != "none" {
			diags.AddAttributeError(path.Root("injection").AtName("mode"), "Invalid value", "mode must be all, selected or none.")
		}
	}
	if v, ok := attrs["variable_name"].(types.String); ok && !v.IsNull() && !v.IsUnknown() && !variableName.MatchString(v.ValueString()) {
		diags.AddAttributeError(path.Root("injection").AtName("variable_name"), "Invalid value",
			"variable_name must be upper-case letters, digits and underscores, starting with a letter.")
	}
}
