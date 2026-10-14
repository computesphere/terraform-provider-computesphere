package provider

import (
	"testing"

	csv2 "github.com/computesphere/computesphere-go"
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
