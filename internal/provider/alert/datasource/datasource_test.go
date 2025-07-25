package provider_test

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/config"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"

	th "github.com/computesphere/terraform-provider-computesphere/internal/provider/testhelpers"
)

func TestAccAlertDataSource_basic(t *testing.T) {
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: th.SetupRecordingProvider(t, "alert_datasource_cassette"),
		Steps: []resource.TestStep{
			{
				ConfigFile: config.StaticFile("./testdata/alert_datasource.tf"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("data.computesphere_alert.example", "id"),
					resource.TestCheckResourceAttr("data.computesphere_alert.example", "alert_type", "cpu"),
				),
			},
		},
	})
}
