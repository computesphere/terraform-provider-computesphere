package dataproduct_test

import (
	"context"
	"strings"
	"testing"

	"github.com/computesphere/terraform-provider-computesphere/internal/provider/common/dataproduct"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
)

var testSchema = schema.Schema{Attributes: map[string]schema.Attribute{
	"name":           schema.StringAttribute{Required: true},
	"environment_id": schema.StringAttribute{Required: true},
	"engine":         schema.StringAttribute{Optional: true},
	"version":        schema.StringAttribute{Optional: true},
}}

func raw(name, env, engine, version string) tftypes.Value {
	t := tftypes.Object{AttributeTypes: map[string]tftypes.Type{
		"name": tftypes.String, "environment_id": tftypes.String, "engine": tftypes.String, "version": tftypes.String,
	}}
	return tftypes.NewValue(t, map[string]tftypes.Value{
		"name": tftypes.NewValue(tftypes.String, name), "environment_id": tftypes.NewValue(tftypes.String, env),
		"engine": tftypes.NewValue(tftypes.String, engine), "version": tftypes.NewValue(tftypes.String, version),
	})
}

// Update refuses each immutable attribute that differs from state, so a
// value that was unknown at plan time can't slip through to the API.
func TestRefuseChangeInUpdate(t *testing.T) {
	ctx := context.Background()
	state := tfsdk.State{Schema: testSchema, Raw: raw("app", "env-1", "valkey", "8.0")}
	for _, tc := range []struct {
		attr string
		plan tftypes.Value
	}{
		{"name", raw("app-2", "env-1", "valkey", "8.0")},
		{"environment_id", raw("app", "env-2", "valkey", "8.0")},
		{"engine", raw("app", "env-1", "other", "8.0")},
		{"version", raw("app", "env-1", "valkey", "8.1")},
	} {
		req := resource.UpdateRequest{Plan: tfsdk.Plan{Schema: testSchema, Raw: tc.plan}, State: state}
		resp := &resource.UpdateResponse{}
		if dataproduct.RefuseChangeInUpdate(ctx, req, resp, "cache", "name", "environment_id", "engine", "version") {
			t.Errorf("%s: change allowed", tc.attr)
			continue
		}
		if got := resp.Diagnostics.Errors()[0].Summary(); !strings.HasPrefix(got, tc.attr+" can't change") {
			t.Errorf("%s: summary %q", tc.attr, got)
		}
	}
	req := resource.UpdateRequest{Plan: tfsdk.Plan{Schema: testSchema, Raw: raw("app", "env-1", "valkey", "8.0")}, State: state}
	if !dataproduct.RefuseChangeInUpdate(ctx, req, &resource.UpdateResponse{}, "cache", "name", "environment_id", "engine", "version") {
		t.Error("an unchanged plan was refused")
	}
}
