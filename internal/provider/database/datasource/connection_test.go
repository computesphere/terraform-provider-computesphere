package provider_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/providerserver"
	"github.com/hashicorp/terraform-plugin-go/tfprotov6"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"

	"github.com/computesphere/terraform-provider-computesphere/internal/provider"
)

const connDB = "99999999-9999-4999-8999-999999999999"

// The connection data source reads the routes; reveal is sent only when set,
// and public routes are null until public access is on.
func TestAccDatabaseConnectionDataSource(t *testing.T) {
	var revealSeen []string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet || r.URL.Path != "/v2/databases/"+connDB+"/connection" {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		revealSeen = append(revealSeen, r.URL.Query().Get("reveal"))
		pw := "••••••••"
		if r.URL.Query().Get("reveal") == "true" {
			pw = "s3cret"
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"object":"database_connection","database_id":"` + connDB + `","engine":"postgresql",
"pooled":{"label":"For your app","host":"app-db-pool.internal.example","port":6432,"database":"app","username":"app","password":"` + pw + `","uri":"postgresql://app:` + pw + `@app-db-pool.internal.example:6432/app?sslmode=require"},
"direct":{"label":"For tools and migrations","host":"app-db.internal.example","port":5432,"database":"app","username":"app","password":"` + pw + `","uri":"postgresql://app:` + pw + `@app-db.internal.example:5432/app?sslmode=require"},
"revealed":` + map[bool]string{true: "true", false: "false"}[pw == "s3cret"] + `,"sslmode":"require","public":null,"ca_file":"/etc/example/db.crt"}`))
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
	const addr = "data.computesphere_database_connection.example"
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: factories,
		Steps: []resource.TestStep{
			{
				Config: `data "computesphere_database_connection" "example" { database_id = "` + connDB + `" }`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(addr, "engine", "postgresql"),
					resource.TestCheckResourceAttr(addr, "pooled.port", "6432"),
					resource.TestCheckResourceAttr(addr, "direct.host", "app-db.internal.example"),
					resource.TestCheckResourceAttr(addr, "pooled.password", "••••••••"),
					resource.TestCheckResourceAttr(addr, "revealed", "false"),
					resource.TestCheckResourceAttr(addr, "ca_file", "/etc/example/db.crt"),
					resource.TestCheckNoResourceAttr(addr, "public_direct"),
				),
			},
			{
				Config: `data "computesphere_database_connection" "example" {
  database_id = "` + connDB + `"
  reveal      = true
}`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(addr, "pooled.password", "s3cret"),
					resource.TestCheckResourceAttr(addr, "revealed", "true"),
				),
			},
		},
	})
}
