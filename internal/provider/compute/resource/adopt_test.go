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
// version on a worker records it without a rotation or any API write; a
// later version rotates in place.
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
				// A real rotation after that goes in place.
				PreConfig: api.reset,
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
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{checks.ExpectNoReplace()},
				},
				Check: func(*terraformState) error {
					p := api.bodies("PATCH", "")
					img, _ := p[0]["image"].(map[string]any)
					if len(p) != 1 || img["password"] != "registry-secret-2" {
						return fmt.Errorf("rotation change = %v", p)
					}
					return nil
				},
			},
		},
	})
}

// Removing git_repo_id from the configuration keeps the recorded value, so
// adding a different one afterwards is a change, and is refused.
func TestGitRepoID_removeThenAddRefused(t *testing.T) {
	api := newFakeDeploymentsAPI(t)
	name := "computesphere_web_service.app"
	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: th.SetupFakeAPIProvider(t, api),
		Steps: []resource.TestStep{
			{Config: gitWebService("4821")},
			{
				Config: gitWebService(""),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{plancheck.ExpectResourceAction(name, plancheck.ResourceActionNoop)},
				},
				Check: resource.TestCheckResourceAttr(name, "git_repo_id", "4821"),
			},
			{
				Config:      gitWebService("9999"),
				ExpectError: regexp.MustCompile(`git_repo_id can't change on an existing web service`),
			},
		},
	})
}

// A git_repo_id only known at apply (here: from a resource replaced in the
// same run) isn't checked when the plan is made; the apply fails with the
// refusal and nothing reaches the API.
func TestGitRepoID_unknownAtPlanRefusedAtApply(t *testing.T) {
	api := newFakeDeploymentsAPI(t)
	cfg := func(trigger string) string {
		return fmt.Sprintf(`
resource "terraform_data" "repo" {
  triggers_replace = %q
}

resource "computesphere_web_service" "app" {
%s
  git_url           = "group/app"
  git_connection_id = "conn-1"
  git_repo_id       = terraform_data.repo.id
}
`, trigger, ids)
	}
	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: th.SetupFakeAPIProvider(t, api),
		Steps: []resource.TestStep{
			{Config: cfg("a")},
			{
				PreConfig:   api.reset,
				Config:      cfg("b"),
				ExpectError: regexp.MustCompile(`git_repo_id can't change on an existing web service`),
			},
		},
	})
	// The test's own destroy is the only DELETE after the reset.
	if n := api.count("PATCH", "") + api.count("POST", "/deployments") + api.count("PUT", ""); n != 0 {
		t.Fatalf("a refused git_repo_id change reached the API (%d writes)", n)
	}
}

// An imported service reads its real git_repo_id: the same value plans no
// change, and a different one is drift the API can't fix, so it's refused.
func TestGitRepoID_importReadsRealValue(t *testing.T) {
	api := newFakeDeploymentsAPI(t)
	id := api.seedWithRepo("web-service", "4821")
	name := "computesphere_web_service.app"
	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: th.SetupFakeAPIProvider(t, api),
		Steps: []resource.TestStep{
			{
				Config:             gitWebService("4821"),
				ResourceName:       name,
				ImportState:        true,
				ImportStateId:      id,
				ImportStatePersist: true,
				ImportStateCheck: func(s []*terraformInstanceState) error {
					if got := s[0].Attributes["git_repo_id"]; got != "4821" {
						return fmt.Errorf("imported git_repo_id = %q, want the service's 4821", got)
					}
					return nil
				},
			},
			{
				Config:      gitWebService("9999"),
				ExpectError: regexp.MustCompile(`This web service has git_repo_id "4821"`),
			},
			{
				PreConfig: api.reset,
				Config:    gitWebService("4821"),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{checks.ExpectNoReplace()},
				},
				Check: resource.TestCheckResourceAttr(name, "git_repo_id", "4821"),
			},
		},
	})
}

// A static site takes git_repo_id at create (in ci_config, alone) and
// refuses a change to it.
func TestGitRepoID_staticSite(t *testing.T) {
	api := newFakeDeploymentsAPI(t)
	cfg := func(repo string) string {
		return fmt.Sprintf(`
resource "computesphere_static_site" "docs" {
%s
  git_url       = "group/docs"
  build_command = "npm run build"
  git_repo_id   = %q
}
`, ids, repo)
	}
	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: th.SetupFakeAPIProvider(t, api),
		Steps: []resource.TestStep{
			{
				Config: cfg("77"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("computesphere_static_site.docs", "git_repo_id", "77"),
					func(*terraformState) error {
						c := api.bodies("POST", "/deployments")[0]
						checkClosed(t, "static site create", c, createKeys["static-site"])
						ci, _ := c["ci_config"].(map[string]any)
						if len(ci) != 1 || ci["git_repo_id"] != "77" {
							return fmt.Errorf("ci_config = %v", c["ci_config"])
						}
						return nil
					},
				),
			},
			{
				Config:      cfg("78"),
				ExpectError: regexp.MustCompile(`git_repo_id can't change on an existing static site`),
			},
			{Config: cfg("77")},
		},
	})
}
