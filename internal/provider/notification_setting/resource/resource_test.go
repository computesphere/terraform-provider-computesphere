package provider_test

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/config"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"

	th "github.com/computesphere/terraform-provider-computesphere/internal/provider/testhelpers"
)

func TestAccNotificationSettingResource_basic(t *testing.T) {
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: th.SetupRecordingProvider(t, "notification_setting_resource_cassette"),
		Steps: []resource.TestStep{
			{
				ConfigFile: config.StaticFile("./testdata/notification_setting_resource.tf"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("computesphere_notification_setting.example", "id"),
					resource.TestCheckResourceAttr("computesphere_notification_setting.example", "email_enabled", "true"),
				),
			},
		},
	})
}
