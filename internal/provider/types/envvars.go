package types

import (
	"context"
	"net/http"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// WholeSet marks an env-var write as the whole set (replace=true): the maps
// Terraform sends are the configuration, so keys left out are meant to go.
func WholeSet(_ context.Context, req *http.Request) error {
	q := req.URL.Query()
	q.Set("replace", "true")
	req.URL.RawQuery = q.Encode()
	return nil
}

// MapOrEmpty is the map's value, or an empty map when it is null, so that
// sending it clears the variables a removed map held.
func MapOrEmpty(ctx context.Context, m types.Map, diags *diag.Diagnostics) *map[string]string {
	out := map[string]string{}
	if !m.IsNull() && !m.IsUnknown() {
		diags.Append(m.ElementsAs(ctx, &out, false)...)
	}
	return &out
}
