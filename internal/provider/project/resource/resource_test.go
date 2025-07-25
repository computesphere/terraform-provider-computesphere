package provider_test

import (
	"testing"

	"github.com/computesphere/terraform-provider-computesphere/internal/provider/common/checks"
	"github.com/hashicorp/terraform-plugin-testing/config"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/plancheck"

	th "github.com/computesphere/terraform-provider-computesphere/internal/provider/testhelpers"
)

func TestAccProjectResource_basic(t *testing.T) {
	resourceName := "computesphere_project.example"

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: th.SetupRecordingProvider(t, "project_basic_cassette"),
		Steps: []resource.TestStep{
			{
				ConfigFile: config.StaticFile("./testdata/project_basic.tf"),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{
						checks.ExpectNoReplace(),
					},
				},
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "name", "tf-test-project"),
					resource.TestCheckResourceAttr(resourceName, "description", "Test project from acceptance test"),
					resource.TestCheckResourceAttr(resourceName, "plan_name", "MAX"),
					resource.TestCheckResourceAttr(resourceName, "plan_value", "2"),
				),
			},
		},
	})
}
