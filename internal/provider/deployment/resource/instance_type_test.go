package provider_test

import (
	"fmt"
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/plancheck"
)

// instance_type is sent on create and changed in place through scale; a
// size-only change sends no spherelet count. It can't be set with plan_id.
func TestAccDeploymentResource_instanceType(t *testing.T) {
	factories, f := setupFakeDeployAPI(t)
	base := []string{aImage, aPort, aNoWait}
	cfg := func(attrs ...string) string {
		return depConfig(serviceID, envID, append(append([]string{}, base...), attrs...)...)
	}
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: factories,
		Steps: []resource.TestStep{
			{
				Config:      cfg(`instance_type = "sl-1c-2g"`, aPlanID),
				ExpectError: regexp.MustCompile(`Conflicting attributes`),
			},
			{
				PreConfig: f.reset,
				Config:    cfg(`instance_type = "sl-1c-2g"`),
				Check: expectBody(f, "POST /v2/deployments {", func(b map[string]any) error {
					if b["instance_type"] != "sl-1c-2g" {
						return fmt.Errorf("create body instance_type = %v", b["instance_type"])
					}
					return nil
				}),
			},
			{
				PreConfig:        f.reset,
				Config:           cfg(`instance_type = "sl-2c-4g"`),
				ConfigPlanChecks: planAction(plancheck.ResourceActionUpdate),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(depAddr, "instance_type", "sl-2c-4g"),
					expectSent(f, "DELETE ", 0),
					expectBody(f, "POST /v2/deployments/bbbbbbbb-0000-4000-8000-000000000001/scale", func(b map[string]any) error {
						if b["instance_type"] != "sl-2c-4g" || b["skip_redeploy"] != true {
							return fmt.Errorf("scale body = %v, want instance_type sl-2c-4g, skip_redeploy", b)
						}
						if _, ok := b["spherelets"]; ok {
							return fmt.Errorf("scale body = %v: a size-only change sent a spherelet count", b)
						}
						return nil
					}),
				),
			},
		},
	})
}
