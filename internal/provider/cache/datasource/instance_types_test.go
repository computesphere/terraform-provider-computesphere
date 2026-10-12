package provider_test

import (
	"net/http"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"

	th "github.com/computesphere/terraform-provider-computesphere/internal/provider/testhelpers"
)

func TestCacheInstanceTypesDataSource(t *testing.T) {
	var gotQuery string
	api := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet || r.URL.Path != "/v2/caches/instance-types" {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		gotQuery = r.URL.RawQuery
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"object":"list","items":[` +
			`{"object":"cache_instance_type","id":"sc-0.25c-1g","engine":"valkey","name":"Cache 1 GB","vcpu":0.25,"memory_mb":1024,"high_availability":false,"high_availability_type":null,"persistence":true,"monthly_price":{"amount":1500,"currency":"usd"},"price_group":"standard","available":true,"unavailable_reason":null},` +
			`{"object":"cache_instance_type","id":"sc-1c-3g-ha","engine":"valkey","name":"Cache 3 GB, high availability","vcpu":null,"memory_mb":3072,"high_availability":true,"high_availability_type":null,"persistence":true,"monthly_price":null,"price_group":"premium","available":false,"unavailable_reason":"Not offered in this region yet."}` +
			`],"pagination":{},"warnings":[]}`))
	})
	name := "data.computesphere_cache_instance_types.all"
	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: th.SetupFakeAPIProvider(t, api),
		Steps: []resource.TestStep{
			{
				Config: `
data "computesphere_cache_instance_types" "all" {
  environment_id = "33333333-3333-3333-3333-333333333333"
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(name, "instance_types.#", "2"),
					resource.TestCheckResourceAttr(name, "instance_types.0.id", "sc-0.25c-1g"),
					resource.TestCheckResourceAttr(name, "instance_types.0.monthly_price_cents", "1500"),
					resource.TestCheckResourceAttr(name, "instance_types.0.vcpu", "0.25"),
					resource.TestCheckResourceAttr(name, "instance_types.1.high_availability", "true"),
					resource.TestCheckNoResourceAttr(name, "instance_types.1.monthly_price_cents"),
					resource.TestCheckNoResourceAttr(name, "instance_types.1.vcpu"),
					resource.TestCheckResourceAttr(name, "instance_types.1.available", "false"),
					resource.TestCheckResourceAttr(name, "instance_types.1.unavailable_reason", "Not offered in this region yet."),
				),
			},
		},
	})
	if gotQuery != "environment_id=33333333-3333-3333-3333-333333333333" {
		t.Fatalf("query = %q, want the environment_id filter", gotQuery)
	}
}
