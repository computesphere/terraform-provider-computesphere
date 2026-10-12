package provider_test

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"

	"github.com/computesphere/terraform-provider-computesphere/internal/provider/spherelet"
	th "github.com/computesphere/terraform-provider-computesphere/internal/provider/testhelpers"
)

func TestAccSphereletTypesDataSource_region(t *testing.T) {
	factories, fake := th.SetupFakeProvider(t)
	name := "data.computesphere_spherelet_types.us"
	wantRequest := "GET /v2/" + spherelet.ListPath + "?" + spherelet.ListQuery("us-east-1").Encode()
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: factories,
		Steps: []resource.TestStep{
			{
				Config: `data "computesphere_spherelet_types" "us" { region = "us-east-1" }`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(name, "region", "us-east-1"),
					resource.TestCheckResourceAttr(name, "spherelet_types.#", "3"),
					resource.TestCheckTypeSetElemNestedAttrs(name, "spherelet_types.*", map[string]string{
						"id":                  "sl-1c-2g",
						"name":                "Standard 1",
						"family":              "standard",
						"vcpu":                "1",
						"cpu_guaranteed_vcpu": "1",
						"memory_mb":           "2048",
						"monthly_price_usd":   "20",
						"annual_price_usd":    "200",
						"available":           "true",
						"aliases.#":           "1",
						"aliases.0":           "MAX",
					}),
					resource.TestCheckTypeSetElemNestedAttrs(name, "spherelet_types.*", map[string]string{
						"id":                  "sl-0.25c-512m",
						"family":              "flex",
						"vcpu":                "0.25",
						"cpu_guaranteed_vcpu": "0.25",
					}),
					func(*terraform.State) error {
						if fake.Logged(wantRequest) == 0 {
							return fmt.Errorf("no request %q in %v", wantRequest, fake.Requests)
						}
						return nil
					},
				),
			},
		},
	})
}
