package provider_test

import (
	"os"
	"regexp"
	"strings"
	"testing"
	"time"

	cacheresource "github.com/computesphere/terraform-provider-computesphere/internal/provider/cache/resource"
	"github.com/computesphere/terraform-provider-computesphere/internal/provider/common/checks"
	"github.com/hashicorp/terraform-plugin-testing/config"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/plancheck"

	th "github.com/computesphere/terraform-provider-computesphere/internal/provider/testhelpers"
)

const cacheConfig = `
resource "computesphere_cache" "example" {
  name           = "app-cache"
  environment_id = "33333333-3333-3333-3333-333333333333"
  instance_type  = %q
  eviction       = %q
}
`

// Creates a cache against the fake API, resizes it to its high-availability
// twin and changes the eviction policy in place (no replacement), then
// destroys it. The connection route is never called: no password reaches
// state.
func TestCacheResource_lifecycle(t *testing.T) {
	cacheresource.SetPollIntervalForTest(time.Millisecond)
	api := newFakeCacheAPI(t)
	name := "computesphere_cache.example"

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: th.SetupFakeAPIProvider(t, api),
		Steps: []resource.TestStep{
			{
				Config: sprintf(cacheConfig, "sc-0.25c-1g", "allkeys-lru"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(name, "id", fakeCacheID),
					resource.TestCheckResourceAttr(name, "engine", "valkey"),
					resource.TestCheckResourceAttr(name, "version", "8.1"),
					resource.TestCheckResourceAttr(name, "instance_type", "sc-0.25c-1g"),
					resource.TestCheckResourceAttr(name, "high_availability", "false"),
					resource.TestCheckResourceAttr(name, "status", "available"),
					resource.TestCheckResourceAttr(name, "region", "eastus2"),
				),
			},
			{
				Config: sprintf(cacheConfig, "sc-1c-3g-ha", "noeviction"),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{
						checks.ExpectNoReplace(),
						plancheck.ExpectResourceAction(name, plancheck.ResourceActionUpdate),
					},
				},
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(name, "instance_type", "sc-1c-3g-ha"),
					resource.TestCheckResourceAttr(name, "eviction", "noeviction"),
					resource.TestCheckResourceAttr(name, "high_availability", "true"),
					resource.TestCheckResourceAttr(name, "status", "available"),
				),
			},
			{
				ResourceName:      name,
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
	if api.called("GET /v2/caches/" + fakeCacheID + "/connection") {
		t.Fatal("the resource must never read the cache connection (it carries the password)")
	}
}

func TestCacheResource_validation(t *testing.T) {
	api := newFakeCacheAPI(t)
	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: th.SetupFakeAPIProvider(t, api),
		Steps: []resource.TestStep{
			{
				Config:      sprintf(cacheConfig, "sc-0.25c-1g", "volatile-lru"),
				ExpectError: regexp.MustCompile(`eviction must be one of allkeys-lru, noeviction`),
			},
			{
				Config: `
resource "computesphere_cache" "example" {
  name           = "App_Cache"
  environment_id = "33333333-3333-3333-3333-333333333333"
  instance_type  = "sc-0.25c-1g"
}
`,
				ExpectError: regexp.MustCompile(`name must be 1-40 lowercase letters`),
			},
		},
	})
}

// Records against a live API with UPDATE_RECORDINGS=true once SphereCache is
// offered there; replays the cassette otherwise and skips until it exists.
func TestAccCacheResource_basic(t *testing.T) {
	if os.Getenv("UPDATE_RECORDINGS") != "true" {
		if _, err := os.Stat("testdata/cache_basic_cassette.yaml"); err != nil {
			t.Skip("cache_basic_cassette.yaml not recorded yet")
		}
		cacheresource.SetPollIntervalForTest(time.Millisecond)
	}
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: th.SetupRecordingProvider(t, "cache_basic_cassette"),
		Steps: []resource.TestStep{
			{
				ConfigFile: config.StaticFile("./testdata/cache_basic.tf"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("computesphere_cache.example", "status", "available"),
					resource.TestCheckResourceAttrSet("computesphere_cache.example", "id"),
				),
			},
		},
	})
}

// A name, environment, engine or version change on an existing cache is refused at plan time,
// never planned as a replacement that would empty it.
func TestCacheResource_immutableChangesRefused(t *testing.T) {
	cacheresource.SetPollIntervalForTest(time.Millisecond)
	api := newFakeCacheAPI(t)
	api.version = "8.0" // an older version the API created earlier
	base := `
resource "computesphere_cache" "example" {
  name           = "app-cache"
  environment_id = "33333333-3333-3333-3333-333333333333"
  instance_type  = "sc-0.25c-1g"
%s
}
`
	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: th.SetupFakeAPIProvider(t, api),
		Steps: []resource.TestStep{
			{Config: sprintf(base, "")},
			{
				Config:      sprintf(base, `  version = "8.1"`),
				ExpectError: regexp.MustCompile(`version can't change on an existing cache`),
			},
			{
				Config:      strings.Replace(sprintf(base, ""), `name           = "app-cache"`, `name           = "app-cache-renamed"`, 1),
				ExpectError: regexp.MustCompile(`name can't change on an existing cache`),
			},
			{
				Config:      strings.Replace(sprintf(base, ""), "33333333-3333-3333-3333-333333333333", "55555555-5555-5555-5555-555555555555", 1),
				ExpectError: regexp.MustCompile(`environment_id can't change on an\s+existing cache`),
			},
			{
				// Leaving version out keeps the running one: nothing to do.
				Config: sprintf(base, ""),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{plancheck.ExpectResourceAction("computesphere_cache.example", plancheck.ResourceActionNoop)},
				},
			},
		},
	})
	if len(api.creates()) != 1 { // the test's own destroy is the only DELETE
		t.Fatal("an immutable change must never replace the cache")
	}
}

// An environment that is only known at apply (here: replaced in the same
// run) isn't known when the plan is made; the apply fails with the refusal
// and nothing reaches the API.
func TestCacheResource_unknownEnvironmentRefusedAtApply(t *testing.T) {
	cacheresource.SetPollIntervalForTest(time.Millisecond)
	api := newFakeCacheAPI(t)
	cfg := func(trigger string) string {
		return sprintf(`
resource "terraform_data" "env" {
  triggers_replace = %q
}

resource "computesphere_cache" "example" {
  name           = "app-cache"
  environment_id = terraform_data.env.id
  instance_type  = "sc-0.25c-1g"
}
`, trigger)
	}
	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: th.SetupFakeAPIProvider(t, api),
		Steps: []resource.TestStep{
			{Config: cfg("a")},
			{
				Config:      cfg("b"),
				ExpectError: regexp.MustCompile(`environment_id can't change on an\s+existing cache`),
			},
		},
	})
	if api.called("PATCH ") || len(api.creates()) != 1 {
		t.Fatal("a refused environment change must not reach the API")
	}
}
