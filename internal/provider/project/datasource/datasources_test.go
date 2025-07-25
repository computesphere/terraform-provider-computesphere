package provider_test

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/config"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"

	th "github.com/computesphere/terraform-provider-computesphere/internal/provider/testhelpers"
)

func TestAccProjectDatasources_basic(t *testing.T) {
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: th.SetupRecordingProvider(t, "projects_datasource_cassette"),
		Steps: []resource.TestStep{
			{
				ConfigFile: config.StaticFile("./testdata/projects_datasource.tf"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckTypeSetElemAttr("data.computesphere_projects.all", "projects.*.name", "tf-datasource-list-1"),
					resource.TestCheckTypeSetElemAttr("data.computesphere_projects.all", "projects.*.name", "tf-datasource-list-2"),
				),
			},
		},
	})
}
