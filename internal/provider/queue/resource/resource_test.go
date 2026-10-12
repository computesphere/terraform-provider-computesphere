package provider_test

import (
	"os"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/computesphere/terraform-provider-computesphere/internal/provider/common/checks"
	queueresource "github.com/computesphere/terraform-provider-computesphere/internal/provider/queue/resource"
	"github.com/hashicorp/terraform-plugin-testing/config"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/plancheck"

	th "github.com/computesphere/terraform-provider-computesphere/internal/provider/testhelpers"
)

const queueConfig = `
resource "computesphere_queue" "example" {
  name           = "app-queue"
  environment_id = "33333333-3333-3333-3333-333333333333"
  instance_type  = %q
}
`

// Creates a broker against the fake API, moves it to its three-member twin in
// place (no replacement), imports it and destroys it. The connection route
// is never called: no password reaches state.
func TestQueueResource_lifecycle(t *testing.T) {
	queueresource.SetPollIntervalForTest(time.Millisecond)
	api := newFakeQueueAPI(t)
	name := "computesphere_queue.example"

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: th.SetupFakeAPIProvider(t, api),
		Steps: []resource.TestStep{
			{
				Config: sprintf(queueConfig, "sq-1c-2g-s20g"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(name, "id", fakeQueueID),
					resource.TestCheckResourceAttr(name, "engine", "rabbitmq"),
					resource.TestCheckResourceAttr(name, "version", "4.3"),
					resource.TestCheckResourceAttr(name, "nodes", "1"),
					resource.TestCheckResourceAttr(name, "status", "available"),
				),
			},
			{
				Config: sprintf(queueConfig, "sq-1c-2g-s20g-ha"),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{
						checks.ExpectNoReplace(),
						plancheck.ExpectResourceAction(name, plancheck.ResourceActionUpdate),
					},
				},
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(name, "instance_type", "sq-1c-2g-s20g-ha"),
					resource.TestCheckResourceAttr(name, "nodes", "3"),
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
	if api.called("GET /v2/queues/" + fakeQueueID + "/connection") {
		t.Fatal("the resource must never read the broker connection (it carries the password)")
	}
}

func TestQueueResource_validation(t *testing.T) {
	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: th.SetupFakeAPIProvider(t, newFakeQueueAPI(t)),
		Steps: []resource.TestStep{
			{
				Config: `
resource "computesphere_queue" "example" {
  name           = "app-queue"
  environment_id = "33333333-3333-3333-3333-333333333333"
  instance_type  = "sq-1c-2g-s20g"
  version        = "3.13"
}
`,
				ExpectError: regexp.MustCompile(`version must be one of 4.3`),
			},
		},
	})
}

// Records against a live API with UPDATE_RECORDINGS=true once SphereQueue is
// offered there; replays the cassette otherwise and skips until it exists.
func TestAccQueueResource_basic(t *testing.T) {
	if os.Getenv("UPDATE_RECORDINGS") != "true" {
		if _, err := os.Stat("testdata/queue_basic_cassette.yaml"); err != nil {
			t.Skip("queue_basic_cassette.yaml not recorded yet")
		}
		queueresource.SetPollIntervalForTest(time.Millisecond)
	}
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: th.SetupRecordingProvider(t, "queue_basic_cassette"),
		Steps: []resource.TestStep{
			{
				ConfigFile: config.StaticFile("./testdata/queue_basic.tf"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("computesphere_queue.example", "status", "available"),
					resource.TestCheckResourceAttrSet("computesphere_queue.example", "id"),
				),
			},
		},
	})
}

// A name, environment, engine or version change on an existing broker is refused at plan
// time, never planned as a replacement that would drop its messages.
func TestQueueResource_immutableChangesRefused(t *testing.T) {
	queueresource.SetPollIntervalForTest(time.Millisecond)
	api := newFakeQueueAPI(t)
	api.version = "4.2"
	base := `
resource "computesphere_queue" "example" {
  name           = "app-queue"
  environment_id = "33333333-3333-3333-3333-333333333333"
  instance_type  = "sq-1c-2g-s20g"
%s
}
`
	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: th.SetupFakeAPIProvider(t, api),
		Steps: []resource.TestStep{
			{Config: sprintf(base, "")},
			{
				Config:      sprintf(base, `  version = "4.3"`),
				ExpectError: regexp.MustCompile(`version can't change on an existing broker`),
			},
			{
				Config:      strings.Replace(sprintf(base, ""), `name           = "app-queue"`, `name           = "app-queue-renamed"`, 1),
				ExpectError: regexp.MustCompile(`name can't change on an existing broker`),
			},
			{
				Config:      strings.Replace(sprintf(base, ""), "33333333-3333-3333-3333-333333333333", "55555555-5555-5555-5555-555555555555", 1),
				ExpectError: regexp.MustCompile(`environment_id can't change on an\s+existing broker`),
			},
			{Config: sprintf(base, "")},
		},
	})
	if len(api.creates()) != 1 { // the test's own destroy is the only DELETE
		t.Fatal("an immutable change must never replace the broker")
	}
}
