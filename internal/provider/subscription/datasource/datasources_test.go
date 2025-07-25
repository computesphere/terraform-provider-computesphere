package provider_test

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/config"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"

	th "github.com/computesphere/terraform-provider-computesphere/internal/provider/testhelpers"
)

func TestAccSubscriptionsDatasources_basic(t *testing.T) {
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: th.SetupRecordingProvider(t, "subscriptions_datasource_cassette"),
		Steps: []resource.TestStep{
			{
				ConfigFile: config.StaticFile("./testdata/subscriptions_datasource.tf"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("data.computesphere_subscriptions.example", "subscriptions.#"),
				),
			},
		},
	})
}
