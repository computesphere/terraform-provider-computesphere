package provider

import (
	"context"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
)

// validate runs ValidateConfig on a config with the given string attributes
// set and every other attribute null, and returns the error summaries and
// details.
func validate(t *testing.T, attrs map[string]string) []string {
	t.Helper()
	ctx := context.Background()
	r := &DatabaseResource{}
	var sr resource.SchemaResponse
	r.Schema(ctx, resource.SchemaRequest{}, &sr)
	typ, ok := sr.Schema.Type().TerraformType(ctx).(tftypes.Object)
	if !ok {
		t.Fatal("the schema isn't an object")
	}
	vals := map[string]tftypes.Value{}
	for name, at := range typ.AttributeTypes {
		if v, ok := attrs[name]; ok {
			vals[name] = tftypes.NewValue(tftypes.String, v)
			continue
		}
		vals[name] = tftypes.NewValue(at, nil)
	}
	req := resource.ValidateConfigRequest{Config: tfsdk.Config{Schema: sr.Schema, Raw: tftypes.NewValue(typ, vals)}}
	var resp resource.ValidateConfigResponse
	r.ValidateConfig(ctx, req, &resp)
	var out []string
	for _, d := range resp.Diagnostics.Errors() {
		out = append(out, d.Summary()+": "+d.Detail())
	}
	return out
}

func TestValidateConfigDocumentDB(t *testing.T) {
	base := map[string]string{"name": "catalog", "environment_id": "e", "instance_type": "docdb-free"}
	with := func(kv ...string) map[string]string {
		m := map[string]string{}
		for k, v := range base {
			m[k] = v
		}
		for i := 0; i+1 < len(kv); i += 2 {
			m[kv[i]] = kv[i+1]
		}
		return m
	}
	for _, c := range []struct {
		name  string
		attrs map[string]string
		want  string // a substring of an error, or "" for none
	}{
		{"documentdb accepted", with("engine", "documentdb"), ""},
		{"documentdb, version unset", with("engine", "documentdb"), ""},
		{"documentdb, version 18", with("engine", "documentdb", "version", "18"), ""},
		{"documentdb, version 17", with("engine", "documentdb", "version", "17"), "version must be 18"},
		{"another engine refused", with("engine", "mongodb"), "engine must be one of postgresql, documentdb"},
		{"plan on documentdb refused", map[string]string{"name": "catalog", "environment_id": "e", "engine": "documentdb", "plan": "standard"}, "not available on the documentdb engine"},
		{"postgresql, version 16", with("engine", "postgresql", "version", "16", "instance_type", "pg-1c-2g"), ""},
	} {
		errs := validate(t, c.attrs)
		joined := strings.Join(errs, "\n")
		switch {
		case c.want == "" && len(errs) > 0:
			t.Errorf("%s: unexpected errors:\n%s", c.name, joined)
		case c.want != "" && !strings.Contains(joined, c.want):
			t.Errorf("%s: want an error containing %q, got:\n%s", c.name, c.want, joined)
		}
	}
}
