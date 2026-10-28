package provider_test

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/config"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"

	th "github.com/computesphere/terraform-provider-computesphere/internal/provider/testhelpers"
)

func TestAccRegionDataSource_basic(t *testing.T) {
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: th.SetupRecordingProvider(t, "region_datasource_cassette"),
		Steps: []resource.TestStep{
			{
				ConfigFile: config.StaticFile("./testdata/region_datasource.tf"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("data.computesphere_region.default", "name", "us-east-1"),
					// An API that publishes no addresses: both lists are null.
					resource.TestCheckNoResourceAttr("data.computesphere_region.default", "outbound_ips.#"),
					resource.TestCheckNoResourceAttr("data.computesphere_region.default", "build_outbound_ips.#"),
				),
			},
		},
	})
}

// The region's outbound and build addresses, each under its own attribute
// (GET /v2/regions as the API answers it once both are published).
func TestAccRegionDataSource_addresses(t *testing.T) {
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: th.SetupRecordingProvider(t, "region_addresses_cassette"),
		Steps: []resource.TestStep{
			{
				ConfigFile: config.StaticFile("./testdata/region_datasource.tf"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("data.computesphere_region.default", "outbound_ips.#", "1"),
					resource.TestCheckResourceAttr("data.computesphere_region.default", "outbound_ips.0", "192.0.2.0/31"),
					resource.TestCheckResourceAttr("data.computesphere_region.default", "build_outbound_ips.#", "1"),
					resource.TestCheckResourceAttr("data.computesphere_region.default", "build_outbound_ips.0", "198.51.100.0/31"),
				),
			},
		},
	})
}
