package provider_test

import (
	"fmt"
	"regexp"
	"strings"
	"testing"

	"github.com/computesphere/terraform-provider-computesphere/internal/provider/common/checks"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/plancheck"
	"github.com/hashicorp/terraform-plugin-testing/tfversion"

	th "github.com/computesphere/terraform-provider-computesphere/internal/provider/testhelpers"
)

var writeOnlyVersions = []tfversion.TerraformVersionCheck{tfversion.SkipBelow(tfversion.Version1_11_0)}

func privateImageConfig(res, password string, version int, extra string) string {
	return fmt.Sprintf(`
resource %q "app" {
%s
  image                     = "registry.example.test/app:1"
  image_type                = "private"
  image_provider            = "docker"
  image_username            = "deploy"
  image_url                 = "registry.example.test"
  image_password            = %q
  image_password_wo_version = %d
%s
}
`, res, ids, password, version, extra)
}

// The registry password is write-only: sent on create, never in state, and
// a rotation (a new image_password_wo_version) applies in place through the
// change body's image object.
func TestWebService_passwordRotatesInPlace(t *testing.T) {
	api := newFakeDeploymentsAPI(t)
	name := "computesphere_web_service.app"
	resource.UnitTest(t, resource.TestCase{
		TerraformVersionChecks:   writeOnlyVersions,
		ProtoV6ProviderFactories: th.SetupFakeAPIProvider(t, api),
		Steps: []resource.TestStep{
			{
				Config: privateImageConfig("computesphere_web_service", "first-secret", 1, ""),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckNoResourceAttr(name, "image_password"),
					resource.TestCheckResourceAttr(name, "image_password_wo_version", "1"),
					func(_ *terraformState) error {
						c := api.bodies("POST", "/deployments")[0]
						if c["image_password"] != "first-secret" {
							return fmt.Errorf("create body = %v, want the password sent", c)
						}
						return nil
					},
				),
			},
			{
				PreConfig: api.reset,
				Config:    privateImageConfig("computesphere_web_service", "second-secret", 2, ""),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{
						checks.ExpectNoReplace(),
						plancheck.ExpectResourceAction(name, plancheck.ResourceActionUpdate),
					},
				},
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckNoResourceAttr(name, "image_password"),
					func(_ *terraformState) error {
						p := api.bodies("PATCH", "")
						if len(p) != 1 {
							return fmt.Errorf("want 1 change, got %v", p)
						}
						checkClosed(t, "web service change", p[0], changeKeys["web-service"])
						img, _ := p[0]["image"].(map[string]any)
						if img["password"] != "second-secret" {
							return fmt.Errorf("change body = %v, want the new password", p[0])
						}
						return nil
					},
				),
			},
			{
				// A new password without a new version is not sent.
				PreConfig: api.reset,
				Config:    privateImageConfig("computesphere_web_service", "third-secret", 2, ""),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{
						plancheck.ExpectResourceAction(name, plancheck.ResourceActionNoop),
					},
				},
			},
		},
	})
}

// A worker and a cron job rotate the registry password in place too: the
// change body carries only the image object, with the credentials, and the
// resource is never replaced.
func TestPasswordRotatesInPlace_workerAndCron(t *testing.T) {
	for _, tc := range []struct{ res, apiType, extra string }{
		{"computesphere_worker", "background-worker", `  command = "bin/worker"`},
		{"computesphere_cron_job", "cron-job", "  schedule = \"0 2 * * *\"\n  command  = \"bin/report\""},
	} {
		t.Run(tc.res, func(t *testing.T) {
			api := newFakeDeploymentsAPI(t)
			name := tc.res + ".app"
			resource.UnitTest(t, resource.TestCase{
				TerraformVersionChecks:   writeOnlyVersions,
				ProtoV6ProviderFactories: th.SetupFakeAPIProvider(t, api),
				Steps: []resource.TestStep{
					{
						Config: privateImageConfig(tc.res, "first-secret", 1, tc.extra),
						Check:  resource.TestCheckNoResourceAttr(name, "image_password"),
					},
					{
						PreConfig: api.reset,
						Config:    privateImageConfig(tc.res, "second-secret", 2, tc.extra),
						ConfigPlanChecks: resource.ConfigPlanChecks{
							PreApply: []plancheck.PlanCheck{
								checks.ExpectNoReplace(),
								plancheck.ExpectResourceAction(name, plancheck.ResourceActionUpdate),
							},
						},
						Check: resource.ComposeAggregateTestCheckFunc(
							resource.TestCheckNoResourceAttr(name, "image_password"),
							func(_ *terraformState) error {
								p := api.bodies("PATCH", "")
								if len(p) != 1 {
									return fmt.Errorf("want 1 change, got %v", p)
								}
								checkClosed(t, tc.res+" change", p[0], changeKeys[tc.apiType])
								if len(p[0]) != 2 {
									return fmt.Errorf("change body = %v, want only type and image", p[0])
								}
								img, _ := p[0]["image"].(map[string]any)
								want := map[string]any{"name": "registry.example.test/app:1", "type": "private",
									"provider": "docker", "username": "deploy", "password": "second-secret",
									"url": "registry.example.test"}
								if fmt.Sprint(img) != fmt.Sprint(want) {
									return fmt.Errorf("image = %v, want %v", img, want)
								}
								return nil
							},
						),
					},
					{
						// The same version with a new value plans nothing.
						Config: privateImageConfig(tc.res, "third-secret", 2, tc.extra),
						ConfigPlanChecks: resource.ConfigPlanChecks{
							PreApply: []plancheck.PlanCheck{
								plancheck.ExpectResourceAction(name, plancheck.ResourceActionNoop),
							},
						},
					},
				},
			})
			if n := api.count("POST", "/deployments"); n != 0 {
				t.Fatalf("the rotation re-created the %s", tc.res)
			}
		})
	}
}

// Until a type's backend takes image changes, the API answers 422 to an
// image or password change. The apply fails and state keeps the old image
// and password version: going back to the old configuration plans nothing.
func TestImageChangeRejected_stateUnchanged(t *testing.T) {
	for _, tc := range []struct{ res, extra string }{
		{"computesphere_worker", `  command = "bin/worker"`},
		{"computesphere_cron_job", "  schedule = \"0 2 * * *\"\n  command  = \"bin/report\""},
	} {
		t.Run(tc.res, func(t *testing.T) {
			api := newFakeDeploymentsAPI(t)
			name := tc.res + ".app"
			old := privateImageConfig(tc.res, "first-secret", 1, tc.extra)
			noop := resource.ConfigPlanChecks{PreApply: []plancheck.PlanCheck{
				plancheck.ExpectResourceAction(name, plancheck.ResourceActionNoop),
			}}
			resource.UnitTest(t, resource.TestCase{
				TerraformVersionChecks:   writeOnlyVersions,
				ProtoV6ProviderFactories: th.SetupFakeAPIProvider(t, api),
				Steps: []resource.TestStep{
					{Config: old},
					{
						PreConfig:   func() { api.rejectImage = true },
						Config:      strings.Replace(old, "registry.example.test/app:1", "registry.example.test/app:2", 1),
						ExpectError: regexp.MustCompile(`image changes aren't\s+available`),
					},
					{
						Config:           old,
						ConfigPlanChecks: noop,
						Check: resource.ComposeAggregateTestCheckFunc(
							resource.TestCheckResourceAttr(name, "image", "registry.example.test/app:1"),
							resource.TestCheckResourceAttr(name, "image_password_wo_version", "1"),
						),
					},
					{
						Config:      privateImageConfig(tc.res, "second-secret", 2, tc.extra),
						ExpectError: regexp.MustCompile(`image changes aren't\s+available`),
					},
					{
						Config:           old,
						ConfigPlanChecks: noop,
						Check:            resource.TestCheckResourceAttr(name, "image_password_wo_version", "1"),
					},
				},
			})
		})
	}
}

// Moving a cron job to another registry sends the configured password with
// the new image: the API keeps a stored password only for the same
// registry, so leaving it out would carry nothing over.
func TestCronJob_registryChangeSendsPassword(t *testing.T) {
	api := newFakeDeploymentsAPI(t)
	extra := "  schedule = \"0 2 * * *\"\n  command  = \"bin/report\""
	old := privateImageConfig("computesphere_cron_job", "first-secret", 1, extra)
	moved := strings.ReplaceAll(old, "registry.example.test", "other-registry.example.test")
	resource.UnitTest(t, resource.TestCase{
		TerraformVersionChecks:   writeOnlyVersions,
		ProtoV6ProviderFactories: th.SetupFakeAPIProvider(t, api),
		Steps: []resource.TestStep{
			{Config: old},
			{
				PreConfig: api.reset,
				Config:    moved,
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{checks.ExpectNoReplace()},
				},
				Check: func(_ *terraformState) error {
					p := api.bodies("PATCH", "")
					if len(p) != 1 || len(p[0]) != 2 {
						return fmt.Errorf("change = %v, want {type, image}", p)
					}
					img, _ := p[0]["image"].(map[string]any)
					if img["url"] != "other-registry.example.test" || img["password"] != "first-secret" {
						return fmt.Errorf("image = %v, want the new registry with the password", img)
					}
					return nil
				},
			},
		},
	})
}
