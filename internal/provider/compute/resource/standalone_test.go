package provider_test

import (
	"fmt"
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/plancheck"

	th "github.com/computesphere/terraform-provider-computesphere/internal/provider/testhelpers"
)

const envAndProject = `
  environment_id = "33333333-3333-3333-3333-333333333333"
  project_id     = "22222222-2222-2222-2222-222222222222"
`

// Without service_id, name creates the service with the private service:
// the create body names it, and the new service's id is read back.
func TestPrivateService_createsItsService(t *testing.T) {
	api := newFakeDeploymentsAPI(t)
	name := "computesphere_private_service.api"
	cfg := fmt.Sprintf(`
resource "computesphere_private_service" "api" {
  name = "api"
%s
  image = "acme/api:1"
  port  = 8080
}
`, envAndProject)
	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: th.SetupFakeAPIProvider(t, api),
		Steps: []resource.TestStep{
			{
				Config: cfg,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(name, "name", "api"),
					resource.TestCheckResourceAttrSet(name, "service_id"),
					func(*terraformState) error {
						c := api.bodies("POST", "/deployments")[0]
						checkClosed(t, "private service create", c, createKeys["private-service"])
						if _, ok := c["service_id"]; ok || c["name"] != "api" {
							return fmt.Errorf("create body = %v, want name and no service_id", c)
						}
						return nil
					},
				),
			},
			{
				Config: cfg,
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{plancheck.ExpectEmptyPlan()},
				},
			},
		},
	})
}

func TestServiceIDOrName(t *testing.T) {
	for name, tc := range map[string]struct{ attrs, err string }{
		"neither": {"", `Set service_id or name`},
		"both":    {`  name = "api"` + "\n" + `  service_id = "44444444-4444-4444-4444-444444444444"`, `not both`},
	} {
		t.Run(name, func(t *testing.T) {
			resource.UnitTest(t, resource.TestCase{
				ProtoV6ProviderFactories: th.SetupFakeAPIProvider(t, newFakeDeploymentsAPI(t)),
				Steps: []resource.TestStep{{
					Config: fmt.Sprintf(`
resource "computesphere_worker" "w" {
%s
%s
  image   = "acme/worker:1"
  command = "bin/worker"
}`, tc.attrs, envAndProject),
					ExpectError: regexp.MustCompile(tc.err),
				}},
			})
		})
	}
}
