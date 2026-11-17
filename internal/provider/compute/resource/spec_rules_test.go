package provider_test

import (
	"fmt"
	"regexp"
	"testing"

	"github.com/computesphere/terraform-provider-computesphere/internal/provider/common/checks"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/plancheck"

	th "github.com/computesphere/terraform-provider-computesphere/internal/provider/testhelpers"
)

func restrictedWebConfig(network string) string {
	return fmt.Sprintf(`
resource "computesphere_web_service" "app" {
%s
  image = "nginx:1.27"
%s
}
`, ids, network)
}

// Removing network_access opens the service to everyone again and brings
// the default address back, instead of leaving the allowlist in place.
func TestWebService_removingNetworkAccessResetsIt(t *testing.T) {
	api := newFakeDeploymentsAPI(t)
	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: th.SetupFakeAPIProvider(t, api),
		Steps: []resource.TestStep{
			{Config: restrictedWebConfig(`  network_access = {
    type            = "restricted_access"
    whitelisted_ips = ["203.0.113.0/24"]
  }`)},
			{
				PreConfig: api.reset,
				Config:    restrictedWebConfig(""),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{checks.ExpectNoReplace()},
				},
				Check: func(_ *terraformState) error {
					p := api.bodies("PATCH", "")
					if len(p) != 1 {
						return fmt.Errorf("want 1 change, got %v", p)
					}
					checkClosed(t, "web service change", p[0], changeKeys["web-service"])
					n, _ := p[0]["networking"].(map[string]any)
					if n["type"] != "public_access" || n["default_url_disabled"] != false {
						return fmt.Errorf("change body = %v, want networking reset to public_access", p[0])
					}
					return nil
				},
			},
		},
	})
}

// The default address can't be turned off on a new service: the API needs
// a serving custom domain first.
func TestWebService_refusesDefaultURLOffAtCreate(t *testing.T) {
	api := newFakeDeploymentsAPI(t)
	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: th.SetupFakeAPIProvider(t, api),
		Steps: []resource.TestStep{{
			Config: restrictedWebConfig(`  network_access = {
    type                 = "public_access"
    default_url_disabled = true
  }`),
			ExpectError: regexp.MustCompile(`Can't turn the default address off on a new web service`),
		}},
	})
	if n := api.count("POST", "/deployments"); n != 0 {
		t.Fatalf("a refused plan created %d deployments", n)
	}
}

// A cron job needs schedule and command, and a static site git_url and
// build_command: the API refuses a create without them.
func TestRequiredAtCreate(t *testing.T) {
	for name, cfg := range map[string]string{
		"cron job without command": fmt.Sprintf(`
resource "computesphere_cron_job" "c" {
%s
  image    = "acme/job:1"
  schedule = "0 3 * * *"
}`, ids),
		"static site without build_command": fmt.Sprintf(`
resource "computesphere_static_site" "s" {
%s
  git_url = "acme/docs"
}`, ids),
	} {
		t.Run(name, func(t *testing.T) {
			resource.UnitTest(t, resource.TestCase{
				ProtoV6ProviderFactories: th.SetupFakeAPIProvider(t, newFakeDeploymentsAPI(t)),
				Steps: []resource.TestStep{{
					Config:      cfg,
					ExpectError: regexp.MustCompile(`Missing required argument`),
				}},
			})
		})
	}
}
