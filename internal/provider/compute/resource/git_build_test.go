package provider_test

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"

	th "github.com/computesphere/terraform-provider-computesphere/internal/provider/testhelpers"
)

func gitBuiltWebService(repoID, buildCommand, logLevel string) string {
	line := ""
	if repoID != "" {
		line = fmt.Sprintf("  git_repo_id = %q", repoID)
	}
	return fmt.Sprintf(`
resource "computesphere_web_service" "app" {
%s
  git_url           = "https://github.com/acme/app"
  branch            = "main"
  git_connection_id = "conn-1"
  runtime_type      = "docker"
  root_dir          = "web"
  build_command     = %q
  env_vars          = { LOG_LEVEL = %q }
%s
}
`, ids, buildCommand, logLevel, line)
}

func checkRollouts(api *fakeDeploymentsAPI, deploys, builds int) func(*terraformState) error {
	return func(_ *terraformState) error {
		if n := api.count("POST", "/deploy"); n != deploys {
			return fmt.Errorf("want %d deploys, got %d", deploys, n)
		}
		if n := api.count("POST", "/builds"); n != builds {
			return fmt.Errorf("want %d builds started, got %d", builds, n)
		}
		return nil
	}
}

// A git-built create sends the flat git keys; the API starts the first
// build, which rolls the service out, and refuses a /deploy before it. A
// build settings change is saved without a build, then built once; a
// variables-only change redeploys the built image.
func TestWebService_gitBuilt(t *testing.T) {
	api := newFakeDeploymentsAPI(t)
	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: th.SetupFakeAPIProvider(t, api),
		Steps: []resource.TestStep{
			{
				Config: gitBuiltWebService("", "make", "info"),
				Check: resource.ComposeAggregateTestCheckFunc(
					checkRollouts(api, 0, 0),
					func(_ *terraformState) error {
						c := api.bodies("POST", "/deployments")[0]
						if c["git_url"] != "https://github.com/acme/app" || c["build_source"] != "git" {
							return fmt.Errorf("create body = %v, want the flat git source", c)
						}
						if _, ok := c["ci_config"]; ok {
							return fmt.Errorf("create body = %v: ci_config would drop the flat git source", c)
						}
						return nil
					},
				),
			},
			{
				PreConfig: api.reset,
				Config:    gitBuiltWebService("", "make build", "info"),
				Check: resource.ComposeAggregateTestCheckFunc(
					checkRollouts(api, 0, 1),
					func(_ *terraformState) error {
						p := api.bodies("PATCH", "")
						if len(p) != 1 || p[0]["skip_redeploy"] != true || p[0]["build_config"] == nil {
							return fmt.Errorf("change bodies = %v, want one build_config change saved without a build", p)
						}
						b := api.bodies("POST", "/builds")[0]
						if m, ok := b["manifest"].(map[string]any); !ok || len(m) != 0 {
							return fmt.Errorf("build body = %v, want an empty manifest", b)
						}
						return nil
					},
				),
			},
			{
				PreConfig: api.reset,
				Config:    gitBuiltWebService("", "make build", "debug"),
				Check:     checkRollouts(api, 1, 0),
			},
		},
	})
}

// git_repo_id exists only in ci_config, and a create with ci_config is read
// from it alone: every git field goes there, and the provider starts the
// first build, since the API starts none for that form.
func TestWebService_gitRepoIDCreate(t *testing.T) {
	api := newFakeDeploymentsAPI(t)
	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: th.SetupFakeAPIProvider(t, api),
		Steps: []resource.TestStep{
			{
				Config: gitBuiltWebService("4821", "make", "info"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("computesphere_web_service.app", "git_repo_id", "4821"),
					checkRollouts(api, 0, 1),
					func(_ *terraformState) error {
						c := api.bodies("POST", "/deployments")[0]
						for _, k := range []string{"git_url", "branch", "git_connection_id", "runtime_type", "root_dir",
							"build_command", "build_source"} {
							if _, ok := c[k]; ok {
								return fmt.Errorf("create body = %v: flat %s beside ci_config", c, k)
							}
						}
						ci, _ := c["ci_config"].(map[string]any)
						want := map[string]any{"git_url": "https://github.com/acme/app", "git_branch": "main",
							"git_connection_id": "conn-1", "git_repo_id": "4821", "build_type": "docker",
							"root_dir": "web", "build_script": "make"}
						if fmt.Sprint(ci) != fmt.Sprint(want) {
							return fmt.Errorf("ci_config = %v, want %v", ci, want)
						}
						return nil
					},
				),
			},
		},
	})
}

// A static site's create starts its first build, and its deploy is a build:
// no deploy right after create.
func TestStaticSite_createStartsNoSecondBuild(t *testing.T) {
	api := newFakeDeploymentsAPI(t)
	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: th.SetupFakeAPIProvider(t, api),
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(`
resource "computesphere_static_site" "docs" {
%s
  git_url       = "https://github.com/acme/docs"
  build_command = "npm run build"
}
`, ids),
				Check: checkRollouts(api, 0, 0),
			},
		},
	})
}
