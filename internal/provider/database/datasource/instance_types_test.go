package provider_test

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/config"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"

	th "github.com/computesphere/terraform-provider-computesphere/internal/provider/testhelpers"
)

func TestAccDatabaseInstanceTypesDataSource(t *testing.T) {
	name := "data.computesphere_database_instance_types.all"
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: th.SetupRecordingProvider(t, "database_instance_types_cassette"),
		Steps: []resource.TestStep{
			{
				ConfigFile: config.StaticFile("./testdata/database_instance_types.tf"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(name, "instance_types.#", "21"),
					resource.TestCheckTypeSetElemNestedAttrs(name, "instance_types.*", map[string]string{
						"id":                  "pg-2c-4g",
						"name":                "Standard 2",
						"family":              "standard",
						"monthly_price_cents": "7400",
					}),
					resource.TestCheckTypeSetElemNestedAttrs(name, "instance_types.*", map[string]string{
						"id":     "pg-free",
						"family": "free",
					}),
				),
			},
		},
	})
}
