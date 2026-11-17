package provider_test

import (
	"fmt"
	"testing"

	"github.com/computesphere/terraform-provider-computesphere/internal/provider/common/checks"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/plancheck"

	th "github.com/computesphere/terraform-provider-computesphere/internal/provider/testhelpers"
)

func sizedWorkerConfig(size, vars string) string {
	return fmt.Sprintf(`
resource "computesphere_worker" "app" {
%s
  image         = "acme/worker:1"
  command       = "bin/worker"
  instance_type = %q
  env_vars      = { %s }
}
`, ids, size, vars)
}

// instance_type is sent at create and read back; an alias the API stores as
// its code stays as written. A size change goes through scale alone (no
// spherelet count), and removing a variable sends the whole remaining set.
// Each apply rolls out once.
func TestWorker_instanceType(t *testing.T) {
	api := newFakeDeploymentsAPI(t)
	name := "computesphere_worker.app"
	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: th.SetupFakeAPIProvider(t, api),
		Steps: []resource.TestStep{
			{
				Config: sizedWorkerConfig("FLX", `MODE = "a", KEEP = "k"`),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(name, "instance_type", "FLX"),
					func(_ *terraformState) error {
						c := api.bodies("POST", "/deployments")[0]
						checkClosed(t, "worker create", c, createKeys["background-worker"])
						if c["instance_type"] != "FLX" {
							return fmt.Errorf("create body = %v, want instance_type FLX", c)
						}
						if n := api.count("POST", "/deploy"); n != 1 {
							return fmt.Errorf("want one deploy at create, got %d", n)
						}
						return nil
					},
				),
			},
			{
				// The API stored FLX as its code: no diff.
				Config: sizedWorkerConfig("FLX", `MODE = "a", KEEP = "k"`),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{plancheck.ExpectEmptyPlan()},
				},
			},
			{
				PreConfig: api.reset,
				Config:    sizedWorkerConfig("sl-1c-2g", `KEEP = "k"`),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{
						checks.ExpectNoReplace(),
						plancheck.ExpectResourceAction(name, plancheck.ResourceActionUpdate),
					},
				},
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(name, "instance_type", "sl-1c-2g"),
					func(_ *terraformState) error {
						scales := api.bodies("POST", "/scale")
						if len(scales) != 1 {
							return fmt.Errorf("want one scale, got %v", scales)
						}
						s := scales[0]
						if s["instance_type"] != "sl-1c-2g" || s["skip_redeploy"] != true || len(s) != 2 {
							return fmt.Errorf("scale body = %v, want only instance_type and skip_redeploy", s)
						}
						if n := api.count("PATCH", ""); n != 0 {
							return fmt.Errorf("a size change sent %d change bodies", n)
						}
						vars := api.bodies("PUT", "/env-vars")
						if len(vars) != 1 {
							return fmt.Errorf("want one env-var call, got %v", vars)
						}
						ev, _ := vars[0]["env_vars"].(map[string]any)
						if len(ev) != 1 || ev["KEEP"] != "k" {
							return fmt.Errorf("env-var body = %v, want the whole remaining set", vars[0])
						}
						if n := api.count("POST", "/deploy"); n != 1 {
							return fmt.Errorf("want exactly one deploy for the apply, got %d", n)
						}
						return nil
					},
				),
			},
		},
	})
}
