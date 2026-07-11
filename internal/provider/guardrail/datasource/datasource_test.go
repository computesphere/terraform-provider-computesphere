package provider_test

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/config"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"

	th "github.com/computesphere/terraform-provider-computesphere/internal/provider/testhelpers"
)

func TestAccGuardrailDataSource_basic(t *testing.T) {
	t.Skip("guardrail API is broken: created_by empty-string breaks UUID parse, DELETE 500 (see provider task #42)")
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: th.SetupRecordingProvider(t, "guardrail_datasource_cassette"),
		Steps: []resource.TestStep{
			{
				ConfigFile: config.StaticFile("./testdata/guardrail_datasource.tf"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("data.computesphere_guardrail.example", "id"),
					resource.TestCheckResourceAttr("data.computesphere_guardrail.example", "name", "example-guardrail"),
				),
			},
		},
	})
}
