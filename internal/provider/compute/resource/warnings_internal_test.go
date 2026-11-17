package provider

import (
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/diag"
)

// The API's warnings become Terraform warnings, never errors.
func TestAPIWarnings(t *testing.T) {
	var diags diag.Diagnostics
	apiWarnings([]byte(`{"id":"x","warnings":[{"code":"instance_type_fallback","field":"instance_type",`+
		`"message":"The service's own size was used."},{"code":"unknown_field","field":"region","message":"Not used."}]}`),
		"the new web service", &diags)
	if diags.HasError() || diags.WarningsCount() != 2 {
		t.Fatalf("diags = %v, want 2 warnings", diags)
	}
	w := diags.Warnings()[0]
	if !strings.Contains(w.Summary(), "instance_type") || !strings.Contains(w.Detail(), "instance_type_fallback") {
		t.Fatalf("warning = %q / %q", w.Summary(), w.Detail())
	}

	diags = nil
	apiWarnings([]byte(`{"id":"x"}`), "x", &diags)
	apiWarnings([]byte(`not json`), "x", &diags)
	if len(diags) != 0 {
		t.Fatalf("diags = %v, want none", diags)
	}
}
