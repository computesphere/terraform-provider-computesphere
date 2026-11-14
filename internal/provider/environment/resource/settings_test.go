package provider_test

import (
	"fmt"
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/plancheck"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
)

func settingsConfig(extra string) string {
	return fmt.Sprintf(`resource "computesphere_environment" "example" {
  name       = "production"
  region     = %q
  project_id = %q
%s
}`, envRegion, projectA, extra)
}

// Settings set at create are sent after the create; changes are sent in
// place, only the ones that changed; an empty allowlist makes it public.
func TestAccEnvironmentResource_settings(t *testing.T) {
	factories, f := setupFakeEnvAPI(t)
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: factories,
		Steps: []resource.TestStep{
			{
				PreConfig: f.reset,
				Config: settingsConfig(`  deploy_policy = "approval"
  ip_allowlist  = ["203.0.113.0/24"]`),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(envAddr, "deploy_policy", "approval"),
					resource.TestCheckResourceAttr(envAddr, "ip_allowlist.#", "1"),
					resource.TestCheckResourceAttr(envAddr, "default_url_disabled", "false"),
					resource.TestCheckResourceAttr(envAddr, "kind", "standard"),
					resource.TestCheckResourceAttr(envAddr, "sub_domain", "e-production-1a2b"),
					f.expect(
						`POST /v2/environments {"name":"production","project_id":"`+projectA+`","region":"`+envRegion+`"}`,
						`PATCH /v2/environments/`+fakeEnvID+` {"deploy_policy":"approval"}`,
						`PUT /v2/environments/`+fakeEnvID+`/networking {"type":"restricted_access","whitelisted_ips":["203.0.113.0/24"]}`,
					),
				),
			},
			{
				Config: settingsConfig(`  deploy_policy = "approval"
  ip_allowlist  = ["203.0.113.0/24"]`),
				ConfigPlanChecks: resource.ConfigPlanChecks{PreApply: []plancheck.PlanCheck{plancheck.ExpectEmptyPlan()}},
			},
			{
				PreConfig: f.reset,
				Config: settingsConfig(`  deploy_policy = "auto"
  ip_allowlist  = []`),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(envAddr, "deploy_policy", "auto"),
					resource.TestCheckResourceAttr(envAddr, "network_policy_enabled", "true"),
					resource.TestCheckResourceAttr(envAddr, "ip_allowlist.#", "0"),
					f.expect(
						`PATCH /v2/environments/`+fakeEnvID+` {"deploy_policy":"auto","name":"production"}`,
						`PUT /v2/environments/`+fakeEnvID+`/networking {"type":"public_access"}`,
					),
				),
			},
			{
				// Removing the settings from the configuration leaves them as they are.
				Config:           settingsConfig(""),
				ConfigPlanChecks: resource.ConfigPlanChecks{PreApply: []plancheck.PlanCheck{plancheck.ExpectEmptyPlan()}},
			},
		},
	})
}

// Invalid settings are refused at plan time, before any API call.
func TestAccEnvironmentResource_settingsRefused(t *testing.T) {
	factories, f := setupFakeEnvAPI(t)
	refused := func(extra, pattern string) resource.TestStep {
		return resource.TestStep{PreConfig: f.reset, Config: settingsConfig(extra), ExpectError: regexp.MustCompile(pattern)}
	}
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: factories,
		Steps: []resource.TestStep{
			refused(`  default_url_disabled = true`, `can't be set on a new environment`),
			refused(`  deploy_policy = "manual"`, `deploy_policy must be auto or approval`),
			refused(`  ip_allowlist = ["203.0.113.0/24", "203.0.113.0/24"]`, `listed more than once`),
			refused(`  ip_allowlist = [" 203.0.113.0/24"]`, `leading or trailing spaces`),
			refused(`  ip_allowlist = ["not-an-ip"]`, `isn't an IP address or a CIDR range`),
			{
				Config: settingsConfig(`  ip_allowlist = ["198.51.100.7"]`),
				Check: func(*terraform.State) error {
					if len(f.sent) == 0 {
						return fmt.Errorf("nothing sent")
					}
					return nil
				},
			},
		},
	})
}
