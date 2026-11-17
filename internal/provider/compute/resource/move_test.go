package provider_test

import (
	"fmt"
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/plancheck"
	"github.com/hashicorp/terraform-plugin-testing/tfversion"

	th "github.com/computesphere/terraform-provider-computesphere/internal/provider/testhelpers"
)

func deploymentConfig(typ string) string {
	return fmt.Sprintf(`
resource "computesphere_deployment" "app" {
%s
  type         = %q
  image        = "nginx:1.27"
  port         = 8080
  sphere_count = 2
  env_vars     = { MODE = "a" }
}
`, ids, typ)
}

// A moved block carries a computesphere_deployment over to the per-type
// resource: the plan is a no-op, so the running deployment is neither
// replaced nor changed.
func TestMoveState_fromDeployment(t *testing.T) {
	api := newFakeDeploymentsAPI(t)
	name := "computesphere_web_service.app"
	resource.UnitTest(t, resource.TestCase{
		TerraformVersionChecks:   []tfversion.TerraformVersionCheck{tfversion.SkipBelow(tfversion.Version1_8_0)},
		ProtoV6ProviderFactories: th.SetupFakeAPIProvider(t, api),
		Steps: []resource.TestStep{
			{
				Config: deploymentConfig("web-service"),
			},
			{
				PreConfig: api.reset,
				Config: fmt.Sprintf(`
moved {
  from = computesphere_deployment.app
  to   = computesphere_web_service.app
}

resource "computesphere_web_service" "app" {
%s
  image        = "nginx:1.27"
  port         = 8080
  sphere_count = 2
  env_vars     = { MODE = "a" }
}
`, ids),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{
						plancheck.ExpectResourceAction(name, plancheck.ResourceActionNoop),
					},
				},
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(name, "id", "00000000-0000-4000-8000-000000000001"),
					resource.TestCheckResourceAttr(name, "image", "nginx:1.27"),
					resource.TestCheckResourceAttr(name, "sphere_count", "2"),
					func(_ *terraformState) error {
						for _, m := range []string{"POST", "PATCH", "PUT", "DELETE"} {
							if n := api.count(m, ""); n != 0 {
								return fmt.Errorf("the move sent %d %s requests; it must not touch the deployment", n, m)
							}
						}
						return nil
					},
				),
			},
		},
	})
}

// A deployment can only move to the resource of its own type, and the
// error names the right one.
func TestMoveState_wrongType(t *testing.T) {
	api := newFakeDeploymentsAPI(t)
	resource.UnitTest(t, resource.TestCase{
		TerraformVersionChecks:   []tfversion.TerraformVersionCheck{tfversion.SkipBelow(tfversion.Version1_8_0)},
		ProtoV6ProviderFactories: th.SetupFakeAPIProvider(t, api),
		Steps: []resource.TestStep{
			{
				Config: deploymentConfig("cron-job"),
			},
			{
				Config: fmt.Sprintf(`
moved {
  from = computesphere_deployment.app
  to   = computesphere_web_service.app
}

resource "computesphere_web_service" "app" {
%s
  image = "nginx:1.27"
}
`, ids),
				ExpectError: regexp.MustCompile(`Move it to\s+computesphere_cron_job instead`),
			},
			{
				// Back to the original configuration, so the test can clean up.
				Config: deploymentConfig("cron-job"),
			},
		},
	})
}

// Settings the target type doesn't have are dropped with a warning; the
// move still succeeds.
func TestMoveState_dropsPortForWorker(t *testing.T) {
	api := newFakeDeploymentsAPI(t)
	name := "computesphere_worker.app"
	resource.UnitTest(t, resource.TestCase{
		TerraformVersionChecks:   []tfversion.TerraformVersionCheck{tfversion.SkipBelow(tfversion.Version1_8_0)},
		ProtoV6ProviderFactories: th.SetupFakeAPIProvider(t, api),
		Steps: []resource.TestStep{
			{
				Config: deploymentConfig("background-worker"),
			},
			{
				Config: fmt.Sprintf(`
moved {
  from = computesphere_deployment.app
  to   = computesphere_worker.app
}

resource "computesphere_worker" "app" {
%s
  image        = "nginx:1.27"
  sphere_count = 2
  env_vars     = { MODE = "a" }
}
`, ids),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{
						plancheck.ExpectResourceAction(name, plancheck.ResourceActionNoop),
					},
				},
			},
		},
	})
}
