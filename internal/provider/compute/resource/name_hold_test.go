package provider_test

import (
	"fmt"
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"

	th "github.com/computesphere/terraform-provider-computesphere/internal/provider/testhelpers"
)

// namedCronConfig is a cron job created with its own service, by name.
func namedCronConfig(size string) string {
	return fmt.Sprintf(`
resource "computesphere_cron_job" "nightly" {
  name           = "nightly"
  environment_id = "33333333-3333-3333-3333-333333333333"
  project_id     = "22222222-2222-2222-2222-222222222222"
  image          = "acme/report:1"
  schedule       = "0 2 * * *"
  command        = "bin/report"
  instance_type  = %q
}
`, size)
}

// A replacement under the same name sends its create right after the
// delete, while the API still holds the name: the create is sent again
// until the name is free.
func TestCronJob_replaceWaitsForTheName(t *testing.T) {
	api := newFakeDeploymentsAPI(t)
	api.nameHold = 2
	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: th.SetupFakeAPIProvider(t, api),
		Steps: []resource.TestStep{
			{Config: namedCronConfig("FLX")},
			{
				PreConfig: api.reset,
				Config:    namedCronConfig("sl-1c-2g"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("computesphere_cron_job.nightly", "instance_type", "sl-1c-2g"),
					func(_ *terraformState) error {
						if n := api.count("POST", "/deployments"); n != 3 {
							return fmt.Errorf("sent %d creates; want two refused while the name is held, then one created", n)
						}
						return nil
					},
				),
			},
		},
	})
}

// A 409 for a name this apply didn't delete is a real conflict: it fails at
// once, with the API's error.
func TestCronJob_nameInUseFailsAtOnce(t *testing.T) {
	api := newFakeDeploymentsAPI(t)
	api.taken = true
	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: th.SetupFakeAPIProvider(t, api),
		Steps: []resource.TestStep{
			{Config: namedCronConfig("FLX"), ExpectError: regexp.MustCompile(`still being deleted`)},
		},
	})
	if n := api.count("POST", "/deployments"); n != 1 {
		t.Fatalf("sent %d creates; want one, not retried", n)
	}
}

// A name still held when the wait runs out fails with the API's error.
func TestCronJob_replaceGivesUp(t *testing.T) {
	api := newFakeDeploymentsAPI(t)
	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: th.SetupFakeAPIProvider(t, api),
		Steps: []resource.TestStep{
			{Config: namedCronConfig("FLX")},
			{
				PreConfig:   func() { api.mu.Lock(); api.nameHold = 1 << 30; api.mu.Unlock() },
				Config:      namedCronConfig("sl-1c-2g"),
				ExpectError: regexp.MustCompile(`still being deleted`),
			},
		},
	})
}
