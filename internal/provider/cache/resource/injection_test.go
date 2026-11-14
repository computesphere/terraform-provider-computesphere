package provider_test

import (
	"fmt"
	"regexp"
	"testing"
	"time"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/plancheck"
	"github.com/hashicorp/terraform-plugin-testing/terraform"

	cacheresource "github.com/computesphere/terraform-provider-computesphere/internal/provider/cache/resource"
	th "github.com/computesphere/terraform-provider-computesphere/internal/provider/testhelpers"
)

func injectionConfig(block string) string {
	return `
resource "computesphere_cache" "example" {
  name           = "app-cache"
  environment_id = "33333333-3333-3333-3333-333333333333"
  instance_type  = "sc-0.25c-1g"
` + block + `
}
`
}

// A new cache reads its default injection; a configured one is sent; none
// turns it off; leaving the block out keeps it.
func TestCacheResource_injection(t *testing.T) {
	cacheresource.SetPollIntervalForTest(time.Millisecond)
	api := newFakeCacheAPI(t)
	const name = "computesphere_cache.example"
	empty := resource.ConfigPlanChecks{PostApplyPostRefresh: []plancheck.PlanCheck{plancheck.ExpectEmptyPlan()}}
	lastPut := func(want string) resource.TestCheckFunc {
		return func(*terraform.State) error {
			api.mu.Lock()
			defer api.mu.Unlock()
			if len(api.puts) == 0 || fmt.Sprint(api.puts[len(api.puts)-1]) != want {
				return fmt.Errorf("last injection PUT = %v, want %s", api.puts, want)
			}
			return nil
		}
	}
	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: th.SetupFakeAPIProvider(t, api),
		Steps: []resource.TestStep{
			{
				Config: injectionConfig(""),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(name, "injection.mode", "all"),
					resource.TestCheckResourceAttr(name, "injection.variable_name", "REDIS_URL"),
				),
				ConfigPlanChecks: empty,
			},
			{
				Config: injectionConfig(`  injection = {
    mode          = "selected"
    variable_name = "CACHE_URL"
    service_ids   = ["44444444-4444-4444-4444-444444444444"]
  }`),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(name, "injection.mode", "selected"),
					resource.TestCheckResourceAttr(name, "injection.service_ids.#", "1"),
					lastPut("map[mode:selected service_ids:[44444444-4444-4444-4444-444444444444] variable_name:CACHE_URL]"),
				),
				ConfigPlanChecks: empty,
			},
			{
				// Only the variable name changes: the selected services are kept.
				Config: injectionConfig(`  injection = {
    variable_name = "CACHE_URL_2"
  }`),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(name, "injection.mode", "selected"),
					resource.TestCheckResourceAttr(name, "injection.service_ids.#", "1"),
					lastPut("map[mode:selected service_ids:[44444444-4444-4444-4444-444444444444] variable_name:CACHE_URL_2]"),
				),
			},
			{
				Config: injectionConfig(`  injection = {
    mode          = "none"
    variable_name = "CACHE_URL"
    service_ids   = []
  }`),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(name, "injection.mode", "none"),
					resource.TestCheckResourceAttr(name, "injection.service_ids.#", "0"),
				),
				ConfigPlanChecks: empty,
			},
			{
				// Only the variable name: the current mode is sent with it.
				Config: injectionConfig(`  injection = {
    variable_name = "VALKEY_URL"
  }`),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(name, "injection.mode", "none"),
					resource.TestCheckResourceAttr(name, "injection.variable_name", "VALKEY_URL"),
					lastPut("map[mode:none service_ids:[] variable_name:VALKEY_URL]"),
				),
			},
			{
				Config:      injectionConfig(`  injection = { mode = "selected" }`),
				ExpectError: regexp.MustCompile(`needs service_ids`),
			},
			{
				Config:      injectionConfig(`  injection = { mode = "some" }`),
				ExpectError: regexp.MustCompile(`mode must be all, selected or none`),
			},
			{
				Config:      injectionConfig(`  injection = { variable_name = "redis-url" }`),
				ExpectError: regexp.MustCompile(`upper-case letters, digits and underscores`),
			},
			{
				// Without the block, the injection stays as it is.
				Config:           injectionConfig(""),
				ConfigPlanChecks: resource.ConfigPlanChecks{PreApply: []plancheck.PlanCheck{plancheck.ExpectEmptyPlan()}},
			},
		},
	})
}
