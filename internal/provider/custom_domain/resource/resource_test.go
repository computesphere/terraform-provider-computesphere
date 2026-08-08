package provider_test

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/config"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"

	th "github.com/computesphere/terraform-provider-computesphere/internal/provider/testhelpers"
)

func TestAccCustomDomainResource_basic(t *testing.T) {
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: th.SetupRecordingProvider(t, "custom_domain_resource_cassette"),
		Steps: []resource.TestStep{
			{
				ConfigFile: config.StaticFile("./testdata/custom_domain_resource.tf"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("computesphere_custom_domain.example", "id"),
					resource.TestCheckResourceAttrSet("computesphere_custom_domain.example", "hostname"),
					resource.TestCheckResourceAttrSet("computesphere_custom_domain.example", "status"),
					resource.TestCheckResourceAttr("computesphere_custom_domain.example", "domain", "example.com"),
				),
			},
		},
	})
}
