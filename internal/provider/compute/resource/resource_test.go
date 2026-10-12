package provider_test

import (
	"fmt"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/computesphere/terraform-provider-computesphere/internal/provider/common/checks"
	computeresource "github.com/computesphere/terraform-provider-computesphere/internal/provider/compute/resource"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/plancheck"

	th "github.com/computesphere/terraform-provider-computesphere/internal/provider/testhelpers"
)

const ids = `
  service_id     = "44444444-4444-4444-4444-444444444444"
  environment_id = "33333333-3333-3333-3333-333333333333"
  project_id     = "22222222-2222-2222-2222-222222222222"
`

func init() { computeresource.SetPollIntervalForTest(time.Millisecond) }

func webServiceConfig(image string, port int, envValue string, count int) string {
	return fmt.Sprintf(`
resource "computesphere_web_service" "app" {
%s
  image        = %q
  port         = %d
  sphere_count = %d
  env_vars     = { MODE = %q }
  secret_vars  = { TOKEN = "s3cret" }
  deploy_policy = "auto"

  health_check = {
    enabled = true
    path    = "/healthz"
  }
  autoscaling = {
    enabled         = true
    min_replicas    = 1
    max_replicas    = 3
    cpu_utilization = 70
  }
  network_access = {
    type            = "restricted_access"
    whitelisted_ips = ["203.0.113.0/24"]
  }
}
`, ids, image, port, count, envValue)
}

// A web service is created with its closed create body, its change-only
// deploy policy is applied right after, and a change to port, image, env
// vars and scale applies in place through the closed change body, scale
// and env-var routes.
func TestWebService_lifecycle(t *testing.T) {
	api := newFakeDeploymentsAPI(t)
	name := "computesphere_web_service.app"
	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: th.SetupFakeAPIProvider(t, api),
		Steps: []resource.TestStep{
			{
				Config: webServiceConfig("nginx:1.27", 8080, "a", 1),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet(name, "id"),
					resource.TestCheckResourceAttr(name, "name", "app"),
					resource.TestCheckResourceAttr(name, "status", "running"),
					resource.TestCheckResourceAttr(name, "url", "https://app.example.test"),
					resource.TestCheckResourceAttr(name, "internal_url", "app-private.example.test:8080"),
					func(_ *terraformState) error {
						creates := api.bodies("POST", "/deployments")
						if len(creates) != 1 {
							return fmt.Errorf("want 1 create, got %d", len(creates))
						}
						c := creates[0]
						checkClosed(t, "web service create", c, createKeys["web-service"])
						if c["type"] != "web-service" || c["image"] != "nginx:1.27" || c["build_source"] != "image" {
							return fmt.Errorf("create body = %v", c)
						}
						if _, ok := c["auto_scaling_config"]; !ok {
							return fmt.Errorf("autoscaling not sent on create: %v", c)
						}
						patches := api.bodies("PATCH", "")
						if len(patches) != 1 || patches[0]["deploy_policy"] != "auto" || len(patches[0]) != 2 {
							return fmt.Errorf("post-create change = %v, want only type and deploy_policy", patches)
						}
						checkClosed(t, "web service change", patches[0], changeKeys["web-service"])
						return nil
					},
				),
			},
			{
				PreConfig: api.reset,
				Config:    webServiceConfig("nginx:1.28", 9090, "b", 2),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{
						checks.ExpectNoReplace(),
						plancheck.ExpectResourceAction(name, plancheck.ResourceActionUpdate),
					},
				},
				Check: func(_ *terraformState) error {
					patches := api.bodies("PATCH", "")
					if len(patches) != 1 {
						return fmt.Errorf("want 1 change, got %v", patches)
					}
					p := patches[0]
					checkClosed(t, "web service change", p, changeKeys["web-service"])
					img, _ := p["image"].(map[string]any)
					if p["port"] != float64(9090) || img["name"] != "nginx:1.28" {
						return fmt.Errorf("change body = %v", p)
					}
					if _, ok := p["health_check"]; ok {
						return fmt.Errorf("unchanged health check was sent: %v", p)
					}
					if api.count("POST", "/scale") != 1 || api.count("PUT", "/env-vars") != 1 {
						return fmt.Errorf("want one scale and one env-var call")
					}
					if api.count("POST", "/deployments") != 0 || api.count("DELETE", "") != 0 {
						return fmt.Errorf("an in-place change created or deleted a deployment")
					}
					return nil
				},
			},
		},
	})
}

// A worker's image changes in place through the change body's image
// object, and an image worker's command through build_config.run_command.
func TestWorker_imageChangesInPlace(t *testing.T) {
	api := newFakeDeploymentsAPI(t)
	cfg := func(image string) string {
		return fmt.Sprintf(`
resource "computesphere_worker" "jobs" {
%s
  image   = %q
  command = "bin/worker"
}
`, ids, image)
	}
	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: th.SetupFakeAPIProvider(t, api),
		Steps: []resource.TestStep{
			{
				Config: cfg("acme/worker:1"),
				Check: func(_ *terraformState) error {
					c := api.bodies("POST", "/deployments")[0]
					checkClosed(t, "worker create", c, createKeys["background-worker"])
					if c["command"] != "bin/worker" {
						return fmt.Errorf("create body = %v", c)
					}
					if api.count("PATCH", "") != 0 {
						return fmt.Errorf("no change-only setting was given, so nothing should be patched")
					}
					return nil
				},
			},
			{
				PreConfig: api.reset,
				Config:    cfg("acme/worker:2"),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{
						checks.ExpectNoReplace(),
						plancheck.ExpectResourceAction("computesphere_worker.jobs", plancheck.ResourceActionUpdate),
					},
				},
				Check: func(_ *terraformState) error {
					p := api.bodies("PATCH", "")
					if len(p) != 1 {
						return fmt.Errorf("want 1 change, got %v", p)
					}
					checkClosed(t, "worker change", p[0], changeKeys["background-worker"])
					img, _ := p[0]["image"].(map[string]any)
					if img["name"] != "acme/worker:2" || len(p[0]) != 2 {
						return fmt.Errorf("change body = %v", p[0])
					}
					return nil
				},
			},
			{
				// An image worker's command changes in place too: the change
				// body is build_config.run_command alone.
				PreConfig: api.reset,
				Config:    strings.Replace(cfg("acme/worker:2"), "bin/worker", "bin/worker --fast", 1),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{
						checks.ExpectNoReplace(),
						plancheck.ExpectResourceAction("computesphere_worker.jobs", plancheck.ResourceActionUpdate),
					},
				},
				Check: func(_ *terraformState) error {
					p := api.bodies("PATCH", "")
					if len(p) != 1 {
						return fmt.Errorf("want 1 change, got %v", p)
					}
					checkClosed(t, "worker change", p[0], changeKeys["background-worker"])
					bc, _ := p[0]["build_config"].(map[string]any)
					if len(p[0]) != 2 || len(bc) != 1 || bc["run_command"] != "bin/worker --fast" {
						return fmt.Errorf("change body = %v, want {type, build_config:{run_command}}", p[0])
					}
					if api.count("POST", "/deployments") != 0 || api.count("DELETE", "") != 0 {
						return fmt.Errorf("a command change re-created the worker")
					}
					return nil
				},
			},
		},
	})
}

// A cron job's schedule and command change in place through the change
// body's schedule merge.
func TestCronJob_scheduleChangesInPlace(t *testing.T) {
	api := newFakeDeploymentsAPI(t)
	cfg := func(schedule string) string {
		return fmt.Sprintf(`
resource "computesphere_cron_job" "nightly" {
%s
  image    = "acme/tasks:1"
  schedule = %q
  command  = "bin/report"
  env_vars = { REPORT = "daily" }
}
`, ids, schedule)
	}
	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: th.SetupFakeAPIProvider(t, api),
		Steps: []resource.TestStep{
			{
				Config: cfg("0 2 * * *"),
				Check: func(_ *terraformState) error {
					c := api.bodies("POST", "/deployments")[0]
					checkClosed(t, "cron job create", c, createKeys["cron-job"])
					if c["schedule"] != "0 2 * * *" || c["env_vars"] == nil {
						return fmt.Errorf("create body = %v", c)
					}
					return nil
				},
			},
			{
				PreConfig: api.reset,
				Config:    cfg("CRON_TZ=Europe/London 0 3 * * *"),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{checks.ExpectNoReplace()},
				},
				Check: func(_ *terraformState) error {
					p := api.bodies("PATCH", "")
					if len(p) != 1 {
						return fmt.Errorf("want 1 change, got %v", p)
					}
					checkClosed(t, "cron job change", p[0], changeKeys["cron-job"])
					s, _ := p[0]["schedule"].(map[string]any)
					if s["cron"] != "CRON_TZ=Europe/London 0 3 * * *" || len(s) != 1 {
						return fmt.Errorf("change body = %v", p[0])
					}
					return nil
				},
			},
		},
	})
}

// A static site is always built from git; network access, which only its
// change body takes, is applied right after create, and spa changes in
// place through build_config.
func TestStaticSite_lifecycle(t *testing.T) {
	api := newFakeDeploymentsAPI(t)
	cfg := func(spa bool) string {
		return fmt.Sprintf(`
resource "computesphere_static_site" "docs" {
%s
  git_url           = "acme/docs"
  branch            = "main"
  build_command     = "npm run build"
  publish_directory = "dist"
  spa               = %t
  network_access = {
    type = "public_access"
  }
}
`, ids, spa)
	}
	name := "computesphere_static_site.docs"
	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: th.SetupFakeAPIProvider(t, api),
		Steps: []resource.TestStep{
			{
				Config: cfg(true),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(name, "url", "https://app.example.test"),
					func(_ *terraformState) error {
						c := api.bodies("POST", "/deployments")[0]
						checkClosed(t, "static site create", c, createKeys["static-site"])
						if c["build_source"] != "git" || c["git_url"] != "acme/docs" {
							return fmt.Errorf("create body = %v", c)
						}
						p := api.bodies("PATCH", "")
						if len(p) != 1 || p[0]["networking"] == nil {
							return fmt.Errorf("post-create change = %v, want networking", p)
						}
						checkClosed(t, "static site change", p[0], changeKeys["static-site"])
						return nil
					},
				),
			},
			{
				PreConfig: api.reset,
				Config:    cfg(false),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{checks.ExpectNoReplace()},
				},
				Check: func(_ *terraformState) error {
					p := api.bodies("PATCH", "")
					if len(p) != 1 {
						return fmt.Errorf("want 1 change, got %v", p)
					}
					bc, _ := p[0]["build_config"].(map[string]any)
					if bc["spa"] != false || len(bc) != 1 {
						return fmt.Errorf("change body = %v", p[0])
					}
					return nil
				},
			},
		},
	})
}

func TestPrivateService_create(t *testing.T) {
	api := newFakeDeploymentsAPI(t)
	name := "computesphere_private_service.api"
	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: th.SetupFakeAPIProvider(t, api),
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(`
resource "computesphere_private_service" "api" {
%s
  git_url           = "acme/api"
  git_connection_id = "conn-1"
  git_repo_id       = "4821"
  runtime_type      = "native"
  language          = "go"
  build_command     = "go build -o app ."
  start_command     = "./app"
  port              = 9000
  predeploy_command = "./app migrate"
  build_args        = { GOFLAGS = "-mod=mod" }
}
`, ids),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(name, "internal_url", "app-private.example.test:8080"),
					resource.TestCheckNoResourceAttr(name, "url"),
					func(_ *terraformState) error {
						c := api.bodies("POST", "/deployments")[0]
						checkClosed(t, "private service create", c, createKeys["private-service"])
						if c["build_source"] != "git" || c["predeploy_command"] != "./app migrate" {
							return fmt.Errorf("create body = %v", c)
						}
						ci, _ := c["ci_config"].(map[string]any)
						if len(ci) != 1 || ci["git_repo_id"] != "4821" {
							return fmt.Errorf("ci_config = %v, want only git_repo_id", c["ci_config"])
						}
						return nil
					},
				),
			},
		},
	})
}

// A resource refuses a deployment of another type on import.
func TestImport_wrongType(t *testing.T) {
	api := newFakeDeploymentsAPI(t)
	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: th.SetupFakeAPIProvider(t, api),
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(`
resource "computesphere_cron_job" "nightly" {
%s
  image    = "acme/tasks:1"
  schedule = "0 2 * * *"
  command  = "bin/report"
}
`, ids),
			},
			{
				ResourceName:  "computesphere_web_service.other",
				ImportState:   true,
				ImportStateId: "00000000-0000-4000-8000-000000000001",
				Config: fmt.Sprintf(`
resource "computesphere_cron_job" "nightly" {
%s
  image    = "acme/tasks:1"
  schedule = "0 2 * * *"
  command  = "bin/report"
}

resource "computesphere_web_service" "other" {
%s
}
`, ids, ids),
				ExpectError: regexp.MustCompile(`is a cron-job, not a\s+web-service`),
			},
		},
	})
}
