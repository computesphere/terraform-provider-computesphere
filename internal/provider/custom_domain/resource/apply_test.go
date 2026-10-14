package provider

import (
	"context"
	"testing"

	csv2 "github.com/computesphere/computesphere-go"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/stretchr/testify/assert"
)

// An imported domain has only its ids in state; apply fills the domain from the
// server. A configured domain is kept as written.
func TestApplyFillsDomainAfterImport(t *testing.T) {
	id := "11111111-1111-1111-1111-111111111111"
	d := &csv2.Domain{Id: &id, Domain: "app.example.com"}

	imported := customDomainResourceModel{Domain: types.StringNull()}
	imported.apply(d)
	assert.Equal(t, "app.example.com", imported.Domain.ValueString())

	configured := customDomainResourceModel{Domain: types.StringValue("App.Example.com")}
	configured.apply(d)
	assert.Equal(t, "App.Example.com", configured.Domain.ValueString())
}

// After an import, state holds the API's spelling. A configuration that differs
// only in letter case must not replace the domain; another hostname must.
func TestDomainReplaceIgnoresCase(t *testing.T) {
	ctx := context.Background()
	for _, tc := range []struct {
		state, plan string
		replace     bool
	}{
		{"app.example.com", "App.Example.com", false},
		{"app.example.com", "app.example.com", false},
		{"app.example.com", "www.example.com", true},
	} {
		req := planmodifier.StringRequest{StateValue: types.StringValue(tc.state), PlanValue: types.StringValue(tc.plan)}
		var resp stringplanmodifier.RequiresReplaceIfFuncResponse
		replaceIfHostnameDiffers(ctx, req, &resp)
		assert.Equal(t, tc.replace, resp.RequiresReplace, "%s -> %s", tc.state, tc.plan)
	}
}
