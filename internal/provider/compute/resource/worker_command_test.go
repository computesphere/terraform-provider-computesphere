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

func workerConfig(body string) string {
	return fmt.Sprintf(`
resource "computesphere_worker" "w" {
%s
%s
}
`, ids, body)
}

// changeBodyIs runs a two-step test (create, then change in place) and
// checks the one change body against want.
func changeBodyIs(t *testing.T, before, after string, want map[string]any) {
	t.Helper()
	api := newFakeDeploymentsAPI(t)
	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: th.SetupFakeAPIProvider(t, api),
		Steps: []resource.TestStep{
			{Config: workerConfig(before)},
			{
				PreConfig: api.reset,
				Config:    workerConfig(after),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{
						checks.ExpectNoReplace(),
						plancheck.ExpectResourceAction("computesphere_worker.w", plancheck.ResourceActionUpdate),
					},
				},
				Check: func(*terraformState) error {
					p := api.bodies("PATCH", "")
					if len(p) != 1 {
						return fmt.Errorf("want 1 change, got %v", p)
					}
					checkClosed(t, "worker change", p[0], changeKeys["background-worker"])
					if fmt.Sprint(p[0]) != fmt.Sprint(want) {
						return fmt.Errorf("change body = %v, want %v", p[0], want)
					}
					return nil
				},
			},
		},
	})
}

// (a) An image worker's command goes as run_command alone.
func TestWorkerChange_imageCommand(t *testing.T) {
	changeBodyIs(t,
		"  image   = \"acme/worker:1\"\n  command = \"bin/worker\"",
		"  image   = \"acme/worker:1\"\n  command = \"bin/worker --fast\"",
		map[string]any{"type": "background-worker", "build_config": map[string]any{"run_command": "bin/worker --fast"}})
}

// (b) Switching an image worker to git sends the start command as the run
// command, never run_command: null for the dropped image command.
func TestWorkerChange_imageToGitKeepsStartCommand(t *testing.T) {
	changeBodyIs(t,
		"  image   = \"acme/worker:1\"\n  command = \"bin/worker\"",
		"  git_url           = \"group/worker\"\n  git_connection_id = \"conn-1\"\n  start_command     = \"./worker\"",
		map[string]any{"type": "background-worker", "build_config": map[string]any{
			"git_url": "group/worker", "git_connection_id": "conn-1", "run_command": "./worker"}})
}

// (c) A git worker's start command goes as run_command.
func TestWorkerChange_gitStartCommand(t *testing.T) {
	changeBodyIs(t,
		"  git_url           = \"group/worker\"\n  git_connection_id = \"conn-1\"\n  start_command     = \"./worker\"",
		"  git_url           = \"group/worker\"\n  git_connection_id = \"conn-1\"\n  start_command     = \"./worker --fast\"",
		map[string]any{"type": "background-worker", "build_config": map[string]any{"run_command": "./worker --fast"}})
}

// A worker can't set both command and git_url.
func TestWorker_commandWithGitRefused(t *testing.T) {
	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: th.SetupFakeAPIProvider(t, newFakeDeploymentsAPI(t)),
		Steps: []resource.TestStep{{
			Config:      workerConfig("  git_url = \"group/worker\"\n  command = \"bin/worker\""),
			ExpectError: regexp.MustCompile(`command and git_url can't both be set`),
		}},
	})
}

// (d) Switching an image worker to git with no start command clears the
// run command explicitly, so the image's command isn't carried into the
// git build.
func TestWorkerChange_imageToGitWithoutStartCommand(t *testing.T) {
	changeBodyIs(t,
		"  image   = \"acme/worker:1\"\n  command = \"bin/worker\"",
		"  git_url           = \"group/worker\"\n  git_connection_id = \"conn-1\"",
		map[string]any{"type": "background-worker", "build_config": map[string]any{
			"git_url": "group/worker", "git_connection_id": "conn-1", "run_command": nil}})
}

// Switching a git-built service to an image is refused at plan time: the
// API could apply the image and then refuse the build change.
func TestGitToImageRefused(t *testing.T) {
	for _, tc := range []struct{ res, extra string }{
		{"computesphere_worker", ""},
		{"computesphere_web_service", ""},
		{"computesphere_cron_job", "  schedule = \"0 2 * * *\"\n  command  = \"bin/report\"\n"},
	} {
		t.Run(tc.res, func(t *testing.T) {
			api := newFakeDeploymentsAPI(t)
			cfg := func(src string) string {
				return fmt.Sprintf("resource %q \"x\" {\n%s\n%s%s\n}\n", tc.res, ids, tc.extra, src)
			}
			git := "  git_url           = \"group/app\"\n  git_connection_id = \"conn-1\""
			resource.UnitTest(t, resource.TestCase{
				ProtoV6ProviderFactories: th.SetupFakeAPIProvider(t, api),
				Steps: []resource.TestStep{
					{Config: cfg(git)},
					{
						PreConfig:   api.reset,
						Config:      cfg("  image = \"acme/app:1\""),
						ExpectError: regexp.MustCompile(`Can't switch from git to an image in place`),
					},
					{Config: cfg(git)},
				},
			})
			if n := api.count("PATCH", ""); n != 0 {
				t.Fatalf("a refused switch sent %d changes", n)
			}
		})
	}
}

// An image only known at apply (here: from a resource replaced in the same
// run) on a git-built worker whose git_url is gone is refused at plan time
// too, and no change is sent.
func TestGitToImageRefused_unknownImage(t *testing.T) {
	api := newFakeDeploymentsAPI(t)
	git := `
resource "terraform_data" "img" {
  triggers_replace = "a"
}

resource "computesphere_worker" "x" {
%s
  git_url           = "group/app"
  git_connection_id = "conn-1"
}
`
	toImage := `
resource "terraform_data" "img" {
  triggers_replace = "b"
}

resource "computesphere_worker" "x" {
%s
  image = terraform_data.img.id
}
`
	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: th.SetupFakeAPIProvider(t, api),
		Steps: []resource.TestStep{
			{Config: fmt.Sprintf(git, ids)},
			{
				PreConfig:   api.reset,
				Config:      fmt.Sprintf(toImage, ids),
				PlanOnly:    true,
				ExpectError: regexp.MustCompile(`Can't switch from git to an image in place`),
			},
			{
				Config:      fmt.Sprintf(toImage, ids),
				ExpectError: regexp.MustCompile(`Can't switch from git to an image in place`),
			},
			{Config: fmt.Sprintf(git, ids)},
		},
	})
	if n := api.count("PATCH", ""); n != 0 {
		t.Fatalf("a refused switch sent %d changes", n)
	}
}
