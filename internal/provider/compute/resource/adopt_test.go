package provider_test

import (
	"fmt"
	"regexp"
	"testing"

	"github.com/computesphere/terraform-provider-computesphere/internal/provider/common/checks"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/plancheck"
	"github.com/hashicorp/terraform-plugin-testing/tfversion"

	th "github.com/computesphere/terraform-provider-computesphere/internal/provider/testhelpers"
)

// noWrites fails when the fake API saw any write since its last reset.
func noWrites(api *fakeDeploymentsAPI) func(*terraformState) error {
	return func(*terraformState) error {
		for _, m := range []string{"POST", "PATCH", "PUT", "DELETE"} {
			if n := api.count(m, ""); n != 0 {
				return fmt.Errorf("recording a value sent %d %s requests; it must not call the API", n, m)
			}
		}
		return nil
	}
}

func gitWebService(repoID string) string {
	line := ""
	if repoID != "" {
		line = fmt.Sprintf("  git_repo_id = %q", repoID)
	}
	return fmt.Sprintf(`
resource "computesphere_web_service" "app" {
%s
  git_url           = "group/app"
  git_connection_id = "conn-1"
%s
}
`, ids, line)
}

// A recorded git_repo_id can't change: no change body takes it, and a
// running service is never replaced for it.
func TestGitRepoID_changeRefused(t *testing.T) {
	api := newFakeDeploymentsAPI(t)
	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: th.SetupFakeAPIProvider(t, api),
		Steps: []resource.TestStep{
			{Config: gitWebService("4821")},
			{
				Config:      gitWebService("9999"),
				ExpectError: regexp.MustCompile(`git_repo_id can't change on an existing web service`),
			},
			{Config: gitWebService("4821")},
		},
	})
	if n := api.count("POST", "/deployments"); n != 1 {
		t.Fatalf("want 1 create, got %d", n)
	}
}

// After an import, state has no git_repo_id (the API doesn't return it).
// Adding it to the configuration records it: no replacement, no new
// deployment.
func TestGitRepoID_adoptedAfterImport(t *testing.T) {
	api := newFakeDeploymentsAPI(t)
	id := api.seed("web-service")
	name := "computesphere_web_service.app"
	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: th.SetupFakeAPIProvider(t, api),
		Steps: []resource.TestStep{
			{
				Config:             gitWebService(""),
				ResourceName:       name,
				ImportState:        true,
				ImportStateId:      id,
				ImportStatePersist: true,
			},
			{
				PreConfig: api.reset,
				Config:    gitWebService("4821"),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{
						checks.ExpectNoReplace(),
						plancheck.ExpectResourceAction(name, plancheck.ResourceActionUpdate),
					},
				},
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(name, "id", id),
					resource.TestCheckResourceAttr(name, "git_repo_id", "4821"),
					func(*terraformState) error {
						// The import left git_url and the connection out of
						// state (the API doesn't return them either), so they
						// go out as a build change; the repository id never
						// does, and nothing is created or deleted.
						for _, p := range api.bodies("PATCH", "") {
							checkClosed(t, "web service change", p, changeKeys["web-service"])
							if _, ok := p["ci_config"]; ok {
								return fmt.Errorf("git_repo_id was sent on change: %v", p)
							}
						}
						if n := api.count("POST", "/deployments") + api.count("DELETE", ""); n != 0 {
							return fmt.Errorf("adopting git_repo_id created or deleted a deployment")
						}
						return nil
					},
				),
			},
		},
	})
}

// After a moved block from computesphere_deployment, state has no
// git_repo_id. Adding it records it: no replacement, no API write.
func TestGitRepoID_adoptedAfterMove(t *testing.T) {
	api := newFakeDeploymentsAPI(t)
	name := "computesphere_web_service.app"
	resource.UnitTest(t, resource.TestCase{
		TerraformVersionChecks:   []tfversion.TerraformVersionCheck{tfversion.SkipBelow(tfversion.Version1_8_0)},
		ProtoV6ProviderFactories: th.SetupFakeAPIProvider(t, api),
		Steps: []resource.TestStep{
			{Config: deploymentConfig("web-service")},
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
  git_repo_id  = "4821"
}
`, ids),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{
						plancheck.ExpectResourceAction(name, plancheck.ResourceActionUpdate),
					},
				},
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(name, "git_repo_id", "4821"),
					noWrites(api),
				),
			},
		},
	})
}

// After a move, image_password_wo_version is null in state. Setting a first
// version on a worker records it without a rotation (which its change body
// can't carry) and without an API write.
func TestPasswordVersion_adoptedAfterMove(t *testing.T) {
	api := newFakeDeploymentsAPI(t)
	name := "computesphere_worker.app"
	resource.UnitTest(t, resource.TestCase{
		TerraformVersionChecks:   []tfversion.TerraformVersionCheck{tfversion.SkipBelow(tfversion.Version1_11_0)},
		ProtoV6ProviderFactories: th.SetupFakeAPIProvider(t, api),
		Steps: []resource.TestStep{
			{Config: deploymentConfig("background-worker")},
			{
				PreConfig: api.reset,
				Config: fmt.Sprintf(`
moved {
  from = computesphere_deployment.app
  to   = computesphere_worker.app
}

resource "computesphere_worker" "app" {
%s
  image                     = "nginx:1.27"
  sphere_count              = 2
  env_vars                  = { MODE = "a" }
  image_password            = "registry-secret"
  image_password_wo_version = 1
}
`, ids),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{
						plancheck.ExpectResourceAction(name, plancheck.ResourceActionUpdate),
					},
				},
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(name, "image_password_wo_version", "1"),
					resource.TestCheckNoResourceAttr(name, "image_password"),
					noWrites(api),
				),
			},
			{
				// A real rotation after that is still refused.
				Config: fmt.Sprintf(`
resource "computesphere_worker" "app" {
%s
  image                     = "nginx:1.27"
  sphere_count              = 2
  env_vars                  = { MODE = "a" }
  image_password            = "registry-secret-2"
  image_password_wo_version = 2
}
`, ids),
				ExpectError: regexp.MustCompile(`Registry password can't change in place`),
			},
		},
	})
}
