package provider_test

import (
	"fmt"
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/plancheck"
	"github.com/hashicorp/terraform-plugin-testing/terraform"

	th "github.com/computesphere/terraform-provider-computesphere/internal/provider/testhelpers"
)

const svcName = "computesphere_service.example"

func serviceConfig(attrs string) string {
	return fmt.Sprintf(`
resource "computesphere_service" "example" {
  name       = "example-service"
  project_id = "13316e7e-21d7-43c8-9ee7-69674510ceeb"
  type       = "web-service"
  %s
}
`, attrs)
}

func checkFake(f *th.FakeAPI, prefix string, want int) resource.TestCheckFunc {
	return func(*terraform.State) error {
		if got := f.Logged(prefix); got != want {
			return fmt.Errorf("%d requests starting %q, want %d: %v", got, prefix, want, f.Requests)
		}
		return nil
	}
}

// Create with spherelet_type, then change it: an in-place update.
func TestAccServiceResource_sphereletType(t *testing.T) {
	factories, fake := th.SetupFakeProvider(t)
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: factories,
		Steps: []resource.TestStep{
			{
				Config: serviceConfig(`spherelet_type = "sl-1c-2g"`),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(svcName, "spherelet_type", "sl-1c-2g"),
					resource.TestCheckResourceAttr(svcName, "plan_id", th.FakePlanIDs["sl-1c-2g"]),
					checkFake(fake, `POST /v2/services {"name":"example-service","project_id":"13316e7e-21d7-43c8-9ee7-69674510ceeb","spherelet_type":"sl-1c-2g","type":"web-service"}`, 1),
				),
			},
			{
				Config: serviceConfig(`spherelet_type = "sl-2c-4g"`),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{
						plancheck.ExpectResourceAction(svcName, plancheck.ResourceActionUpdate),
					},
				},
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(svcName, "spherelet_type", "sl-2c-4g"),
					resource.TestCheckResourceAttr(svcName, "plan_id", th.FakePlanIDs["sl-2c-4g"]),
					checkFake(fake, `PATCH /v2/services/`, 1),
					checkFake(fake, `PATCH /v2/services/aaaaaaaa-0000-4000-8000-000000000001 {"spherelet_type":"sl-2c-4g"}`, 1),
				),
			},
		},
	})
}

// An alias in the configuration is sent as written and shows no diff once the
// API answers with the code it stands for.
func TestAccServiceResource_sphereletTypeAlias(t *testing.T) {
	factories, fake := th.SetupFakeProvider(t)
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: factories,
		Steps: []resource.TestStep{
			{
				Config: serviceConfig(`spherelet_type = "max"`),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(svcName, "spherelet_type", "max"),
					resource.TestCheckResourceAttr(svcName, "plan_id", th.FakePlanIDs["sl-1c-2g"]),
					checkFake(fake, `POST /v2/services {"name":"example-service","project_id":"13316e7e-21d7-43c8-9ee7-69674510ceeb","spherelet_type":"max","type":"web-service"}`, 1),
				),
			},
			{
				Config: serviceConfig(`spherelet_type = "max"`),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{plancheck.ExpectEmptyPlan()},
				},
				Check: checkFake(fake, `PATCH `, 0),
			},
		},
	})
}

// The deprecated plan_id still creates the service; spherelet_type is read
// back from the API.
func TestAccServiceResource_planIDDeprecated(t *testing.T) {
	factories, fake := th.SetupFakeProvider(t)
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: factories,
		Steps: []resource.TestStep{
			{
				Config: serviceConfig(fmt.Sprintf(`plan_id = %q`, th.FakePlanIDs["sl-0.25c-512m"])),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(svcName, "plan_id", th.FakePlanIDs["sl-0.25c-512m"]),
					resource.TestCheckResourceAttr(svcName, "spherelet_type", "sl-0.25c-512m"),
					checkFake(fake, `POST /v2/services {"name":"example-service","plan_id":"`+th.FakePlanIDs["sl-0.25c-512m"]+`"`, 1),
				),
			},
		},
	})
}

// spherelet_type and plan_id together fail at plan time, before any request.
func TestAccServiceResource_sphereletTypeConflictsWithPlanID(t *testing.T) {
	factories, fake := th.SetupFakeProvider(t)
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: factories,
		Steps: []resource.TestStep{
			{
				Config:      serviceConfig(fmt.Sprintf("spherelet_type = \"sl-1c-2g\"\n  plan_id = %q", th.FakePlanIDs["sl-1c-2g"])),
				PlanOnly:    true,
				ExpectError: regexp.MustCompile(`Set spherelet_type or the deprecated plan_id, not both`),
			},
		},
		CheckDestroy: checkFake(fake, `POST `, 0),
	})
}
