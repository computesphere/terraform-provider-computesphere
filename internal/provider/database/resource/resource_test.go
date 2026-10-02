package provider_test

import (
	"os"
	"testing"
	"time"

	"github.com/computesphere/terraform-provider-computesphere/internal/provider/common/checks"
	dbresource "github.com/computesphere/terraform-provider-computesphere/internal/provider/database/resource"
	"github.com/hashicorp/terraform-plugin-testing/config"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/plancheck"

	th "github.com/computesphere/terraform-provider-computesphere/internal/provider/testhelpers"
)

// Creates a Standard 1 database, resizes it in place (no replacement) to
// Standard 2 with more storage, and destroys it.
func TestAccDatabaseResource_basic(t *testing.T) {
	if os.Getenv("UPDATE_RECORDINGS") != "true" {
		if _, err := os.Stat("testdata/database_basic_cassette.yaml"); err != nil {
			// Record with UPDATE_RECORDINGS=true against an API that can
			// create databases (see the test comment).
			t.Skip("database_basic_cassette.yaml not recorded yet")
		}
		// Replay: don't sleep between the recorded status polls.
		dbresource.SetPollIntervalForTest(time.Millisecond)
	}
	resourceName := "computesphere_database.example"

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: th.SetupRecordingProvider(t, "database_basic_cassette"),
		Steps: []resource.TestStep{
			{
				ConfigFile: config.StaticFile("./testdata/database_basic.tf"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "name", "tf-test-db"),
					resource.TestCheckResourceAttr(resourceName, "instance_type", "pg-1c-2g"),
					resource.TestCheckResourceAttr(resourceName, "high_availability", "none"),
					resource.TestCheckResourceAttr(resourceName, "storage_gb", "10"),
					resource.TestCheckResourceAttr(resourceName, "engine", "postgresql"),
					resource.TestCheckResourceAttr(resourceName, "status", "available"),
					resource.TestCheckResourceAttrSet(resourceName, "id"),
					resource.TestCheckResourceAttrSet(resourceName, "version"),
				),
			},
			{
				ConfigFile: config.StaticFile("./testdata/database_resized.tf"),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{
						checks.ExpectNoReplace(),
					},
				},
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "instance_type", "pg-2c-4g"),
					resource.TestCheckResourceAttr(resourceName, "storage_gb", "15"),
					resource.TestCheckResourceAttr(resourceName, "status", "available"),
				),
			},
		},
	})
}
