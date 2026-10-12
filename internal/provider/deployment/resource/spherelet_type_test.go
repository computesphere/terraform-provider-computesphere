package provider_test

import (
	"fmt"
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/plancheck"
	"github.com/hashicorp/terraform-plugin-testing/terraform"

	th "github.com/computesphere/terraform-provider-computesphere/internal/provider/testhelpers"
)

const depName = "computesphere_deployment.example"

func deploymentConfig(attrs string) string {
	return fmt.Sprintf(`
resource "computesphere_service" "example" {
  name           = "example-service"
  project_id     = "13316e7e-21d7-43c8-9ee7-69674510ceeb"
  type           = "web-service"
  spherelet_type = "sl-0.25c-512m"
}

resource "computesphere_deployment" "example" {
  service_id     = computesphere_service.example.id
  project_id     = "13316e7e-21d7-43c8-9ee7-69674510ceeb"
  environment_id = "b3669d6a-d89e-4037-873a-65903f3f3ced"
  type           = "web-service"
  image          = "nginx:latest"
  port           = 80
  wait_for_ready = false
  %s
}
`, attrs)
}

func checkDepFake(f *th.FakeAPI, prefix string, want int) resource.TestCheckFunc {
	return func(*terraform.State) error {
		if got := f.Logged(prefix); got != want {
			return fmt.Errorf("%d requests starting %q, want %d: %v", got, prefix, want, f.Requests)
		}
		return nil
	}
}

// A deployment takes its own spherelet_type, changes it in place, keeps an
// alias as written, and inherits the service's type when it sets none.
func TestAccDeploymentResource_sphereletType(t *testing.T) {
	factories, fake := th.SetupFakeProvider(t)
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: factories,
		Steps: []resource.TestStep{
			{
				Config: deploymentConfig(`spherelet_type = "sl-1c-2g"`),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(depName, "spherelet_type", "sl-1c-2g"),
					checkDepFake(fake, `POST /v2/deployments {`, 1),
				),
			},
			{
				Config: deploymentConfig(`spherelet_type = "PWR"`),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{
						plancheck.ExpectResourceAction(depName, plancheck.ResourceActionUpdate),
					},
				},
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(depName, "spherelet_type", "PWR"),
					checkDepFake(fake, `PATCH /v2/deployments/`, 1),
				),
			},
			{
				Config: deploymentConfig(`spherelet_type = "PWR"`),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{plancheck.ExpectEmptyPlan()},
				},
			},
		},
	})
}

func TestAccDeploymentResource_inheritsServiceSphereletType(t *testing.T) {
	factories, _ := th.SetupFakeProvider(t)
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: factories,
		Steps: []resource.TestStep{
			{
				Config: deploymentConfig(""),
				Check:  resource.TestCheckResourceAttr(depName, "spherelet_type", "sl-0.25c-512m"),
			},
		},
	})
}

func TestAccDeploymentResource_sphereletTypeConflictsWithPlanID(t *testing.T) {
	factories, _ := th.SetupFakeProvider(t)
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: factories,
		Steps: []resource.TestStep{
			{
				Config:      deploymentConfig(fmt.Sprintf("spherelet_type = \"sl-1c-2g\"\n  plan_id = %q", th.FakePlanIDs["sl-1c-2g"])),
				PlanOnly:    true,
				ExpectError: regexp.MustCompile(`Set spherelet_type or the deprecated plan_id, not both`),
			},
		},
	})
}
