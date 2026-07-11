package provider_test

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/config"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"

	th "github.com/computesphere/terraform-provider-computesphere/internal/provider/testhelpers"
)

func TestAccGuardrailResource_basic(t *testing.T) {
	t.Skip("guardrail API is broken: created_by empty-string breaks UUID parse, DELETE 500 (see provider task #42)")
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: th.SetupRecordingProvider(t, "guardrail_resource_cassette"),
		Steps: []resource.TestStep{
			{
				ConfigFile: config.StaticFile("./testdata/guardrail_resource.tf"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("computesphere_guardrail.example", "id"),
					resource.TestCheckResourceAttr("computesphere_guardrail.example", "name", "example-guardrail"),
				),
			},
		},
	})
}
