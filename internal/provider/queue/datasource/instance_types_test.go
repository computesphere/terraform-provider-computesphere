package provider_test

import (
	"net/http"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"

	th "github.com/computesphere/terraform-provider-computesphere/internal/provider/testhelpers"
)

func TestQueueInstanceTypesDataSource(t *testing.T) {
	api := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet || r.URL.Path != "/v2/queues/instance-types" {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"object":"list","items":[` +
			`{"object":"queue_instance_type","id":"sq-shared","engine":"rabbitmq","name":"Queue Shared","family":"shared","vcpu":null,"memory_mb":null,"storage_gb":null,"nodes":1,"high_availability":false,"high_availability_type":null,"max_connections":20,"max_queues":10,"monthly_price":{"amount":0,"currency":"usd"},"price_group":"standard","available":true,"unavailable_reason":null},` +
			`{"object":"queue_instance_type","id":"sq-1c-2g-s20g","engine":"rabbitmq","name":"Queue 1","family":"dedicated","vcpu":1,"memory_mb":2048,"storage_gb":20,"nodes":1,"high_availability":false,"high_availability_type":"sq-1c-2g-s20g-ha","max_connections":0,"max_queues":0,"monthly_price":null,"price_group":"standard","available":true,"unavailable_reason":null}` +
			`],"pagination":{},"warnings":[]}`))
	})
	name := "data.computesphere_queue_instance_types.all"
	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: th.SetupFakeAPIProvider(t, api),
		Steps: []resource.TestStep{
			{
				Config: `data "computesphere_queue_instance_types" "all" {}`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(name, "instance_types.#", "2"),
					resource.TestCheckResourceAttr(name, "instance_types.0.family", "shared"),
					resource.TestCheckNoResourceAttr(name, "instance_types.0.memory_mb"),
					resource.TestCheckResourceAttr(name, "instance_types.0.monthly_price_cents", "0"),
					resource.TestCheckResourceAttr(name, "instance_types.1.storage_gb", "20"),
					resource.TestCheckResourceAttr(name, "instance_types.1.high_availability_type", "sq-1c-2g-s20g-ha"),
					resource.TestCheckNoResourceAttr(name, "instance_types.1.monthly_price_cents"),
				),
			},
		},
	})
}
