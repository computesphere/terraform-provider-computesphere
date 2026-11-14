package provider_test

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/providerserver"
	"github.com/hashicorp/terraform-plugin-go/tfprotov6"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"

	"github.com/computesphere/terraform-provider-computesphere/internal/provider"
)

const connID = "99999999-9999-4999-8999-999999999999"

// The queue connection is masked unless reveal is true, and reveal is
// sent only then.
func TestAccQueueConnectionDataSource(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet || r.URL.Path != "/v2/queues/"+connID+"/connection" {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		pw, revealed := "••••••••", "false"
		switch r.URL.Query().Get("reveal") {
		case "true":
			pw, revealed = "s3cret", "true"
		case "":
		default:
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"object":"queue_connection","host":"x.internal.example","port":5671,"username":"app","password":"` + pw +
			`","uri":"` + strings.ReplaceAll("amqps://app:PW@q.internal.example:5671/", "PW", pw) + `","tls":true,"revealed":` + revealed + `,"ca_file":"/etc/example/ca.crt","vhost":"/"}`))
	}))
	t.Cleanup(srv.Close)
	factories := map[string]func() (tfprotov6.ProviderServer, error){
		"computesphere": providerserver.NewProtocol6WithError(provider.New("test",
			provider.WithHost(srv.URL+"/v2"),
			provider.WithAPIToken("some-api-token"),
			provider.WithAccountID("11111111-1111-1111-1111-111111111111"),
			provider.WithHTTPClient(srv.Client()),
		)()),
	}
	const addr = "data.computesphere_queue_connection.example"
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: factories,
		Steps: []resource.TestStep{
			{
				Config: `data "computesphere_queue_connection" "example" { queue_id = "` + connID + `" }`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(addr, "port", "5671"),
					resource.TestCheckResourceAttr(addr, "password", "••••••••"),
					resource.TestCheckResourceAttr(addr, "revealed", "false"),
					resource.TestCheckResourceAttr(addr, "tls", "true"),
					resource.TestCheckResourceAttr(addr, "ca_file", "/etc/example/ca.crt"),
					resource.TestCheckResourceAttr(addr, "vhost", "/"),
				),
			},
			{
				Config: `data "computesphere_queue_connection" "example" {
  queue_id = "` + connID + `"
  reveal = true
}`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(addr, "password", "s3cret"),
					resource.TestCheckResourceAttr(addr, "revealed", "true"),
				),
			},
		},
	})
}
