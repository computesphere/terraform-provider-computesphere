package provider

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// object builds a database object with every attribute null except vals.
func object(t *testing.T, vals map[string]string) tftypes.Value {
	t.Helper()
	typ, ok := Schema(context.Background()).Type().TerraformType(context.Background()).(tftypes.Object)
	require.True(t, ok)
	attrs := map[string]tftypes.Value{}
	for name, at := range typ.AttributeTypes {
		if v, ok := vals[name]; ok {
			attrs[name] = tftypes.NewValue(at, v)
		} else {
			attrs[name] = tftypes.NewValue(at, nil)
		}
	}
	return tftypes.NewValue(typ, attrs)
}

func modify(t *testing.T, self string, config, state map[string]string) types.String {
	t.Helper()
	ctx := context.Background()
	s := Schema(ctx)
	cfg := tfsdk.Config{Schema: s, Raw: object(t, config)}
	st := tfsdk.State{Schema: s, Raw: object(t, state)}
	var cv, sv types.String
	require.False(t, cfg.GetAttribute(ctx, path.Root(self), &cv).HasError())
	require.False(t, st.GetAttribute(ctx, path.Root(self), &sv).HasError())
	req := planmodifier.StringRequest{
		Path: path.Root(self), Config: cfg, State: st,
		ConfigValue: cv, StateValue: sv, PlanValue: types.StringUnknown(),
	}
	resp := &planmodifier.StringResponse{PlanValue: req.PlanValue}
	followsSizing{self: self}.PlanModifyString(ctx, req, resp)
	require.False(t, resp.Diagnostics.HasError())
	return resp.PlanValue
}

var prior = map[string]string{"instance_type": "pg-1c-2g", "high_availability": "none", "plan": "hobby"}

func TestFollowsSizing(t *testing.T) {
	// Nothing sizing-related changed: the unset plan keeps its value.
	got := modify(t, "plan", map[string]string{"instance_type": "pg-1c-2g", "high_availability": "none"}, prior)
	assert.Equal(t, "hobby", got.ValueString())

	// High availability changed: the API recomputes plan.
	got = modify(t, "plan", map[string]string{"instance_type": "pg-1c-2g", "high_availability": "standby"}, prior)
	assert.True(t, got.IsUnknown())

	// Instance type changed: same.
	got = modify(t, "plan", map[string]string{"instance_type": "pg-2c-4g"}, prior)
	assert.True(t, got.IsUnknown())

	// The deprecated plan changed: the unset instance type and high availability follow.
	got = modify(t, "instance_type", map[string]string{"plan": "standard"}, prior)
	assert.True(t, got.IsUnknown())
	got = modify(t, "high_availability", map[string]string{"plan": "standard"}, prior)
	assert.True(t, got.IsUnknown())

	// A sibling dropped from the configuration doesn't count as a change.
	got = modify(t, "plan", map[string]string{"instance_type": "pg-1c-2g"}, prior)
	assert.Equal(t, "hobby", got.ValueString())
}

// On create there is no prior state, and a value set in the configuration is
// never touched.
func TestFollowsSizingLeavesCreateAndConfiguredValues(t *testing.T) {
	ctx := context.Background()
	req := planmodifier.StringRequest{StateValue: types.StringNull(), ConfigValue: types.StringNull(), PlanValue: types.StringUnknown()}
	resp := &planmodifier.StringResponse{PlanValue: req.PlanValue}
	followsSizing{self: "plan"}.PlanModifyString(ctx, req, resp)
	assert.True(t, resp.PlanValue.IsUnknown())

	req = planmodifier.StringRequest{StateValue: types.StringValue("pg-1c-2g"), ConfigValue: types.StringValue("pg-2c-4g"), PlanValue: types.StringValue("pg-2c-4g")}
	resp = &planmodifier.StringResponse{PlanValue: req.PlanValue}
	followsSizing{self: "instance_type"}.PlanModifyString(ctx, req, resp)
	assert.Equal(t, "pg-2c-4g", resp.PlanValue.ValueString())
}
