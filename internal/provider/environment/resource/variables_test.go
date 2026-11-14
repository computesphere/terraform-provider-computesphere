package provider_test

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/providerserver"
	"github.com/hashicorp/terraform-plugin-go/tfprotov6"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/plancheck"
	"github.com/hashicorp/terraform-plugin-testing/terraform"

	"github.com/computesphere/terraform-provider-computesphere/internal/provider"
)

// fakeVarsAPI is one environment's shared variables: PUT replaces the map
// (an empty one only with allow_empty, as the API requires), GET returns it.
type fakeVarsAPI struct {
	mu   sync.Mutex
	vars map[string]string
	puts []map[string]any
}

func (f *fakeVarsAPI) handler() http.Handler {
	write := func(w http.ResponseWriter, code int, v any) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(code)
		_ = json.NewEncoder(w).Encode(v)
	}
	view := func() map[string]any {
		return map[string]any{"object": "environment_variables", "variables": f.vars, "quoted": []string{}, "plain_credentials": []string{}, "warnings": []string{}}
	}
	mux := http.NewServeMux()
	mux.HandleFunc("GET /v2/environments/{eid}/variables", func(w http.ResponseWriter, r *http.Request) {
		f.mu.Lock()
		defer f.mu.Unlock()
		write(w, http.StatusOK, view())
	})
	mux.HandleFunc("PUT /v2/environments/{eid}/variables", func(w http.ResponseWriter, r *http.Request) {
		var body map[string]any
		_ = json.NewDecoder(r.Body).Decode(&body)
		f.mu.Lock()
		defer f.mu.Unlock()
		f.puts = append(f.puts, body)
		vars, _ := body["variables"].(map[string]any)
		if len(vars) == 0 && body["allow_empty"] != true {
			write(w, http.StatusUnprocessableEntity, map[string]any{"status": 422, "title": "allow_empty is required"})
			return
		}
		f.vars = map[string]string{}
		for k, v := range vars {
			f.vars[k] = fmt.Sprint(v)
		}
		write(w, http.StatusOK, view())
	})
	return mux
}

func (f *fakeVarsAPI) check(want map[string]string) resource.TestCheckFunc {
	return func(*terraform.State) error {
		f.mu.Lock()
		defer f.mu.Unlock()
		if fmt.Sprint(f.vars) != fmt.Sprint(want) {
			return fmt.Errorf("stored variables = %v, want %v", f.vars, want)
		}
		return nil
	}
}

func setupFakeVarsAPI(t *testing.T) (map[string]func() (tfprotov6.ProviderServer, error), *fakeVarsAPI) {
	t.Helper()
	f := &fakeVarsAPI{vars: map[string]string{}}
	srv := httptest.NewServer(f.handler())
	t.Cleanup(srv.Close)
	return map[string]func() (tfprotov6.ProviderServer, error){
		"computesphere": providerserver.NewProtocol6WithError(provider.New("test",
			provider.WithHost(srv.URL+"/v2"),
			provider.WithAPIToken("some-api-token"),
			provider.WithAccountID("11111111-1111-1111-1111-111111111111"),
			provider.WithHTTPClient(srv.Client()),
		)()),
	}, f
}

func varsConfig(body string) string {
	return `resource "computesphere_environment_variables" "example" {
  environment_id = "` + fakeEnvID + `"
  variables      = ` + body + `
}`
}

func TestAccEnvironmentVariablesResource(t *testing.T) {
	factories, f := setupFakeVarsAPI(t)
	const addr = "computesphere_environment_variables.example"
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: factories,
		CheckDestroy: func(*terraform.State) error {
			f.mu.Lock()
			defer f.mu.Unlock()
			if len(f.vars) != 0 {
				return fmt.Errorf("variables left after destroy: %v", f.vars)
			}
			return nil
		},
		Steps: []resource.TestStep{
			{
				Config: varsConfig(`{ LOG_LEVEL = "info", REGION = "us" }`),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(addr, "id", fakeEnvID),
					resource.TestCheckResourceAttr(addr, "variables.LOG_LEVEL", "info"),
					f.check(map[string]string{"LOG_LEVEL": "info", "REGION": "us"}),
				),
			},
			{
				// A variable added elsewhere shows as a change that removes it.
				PreConfig: func() { f.mu.Lock(); f.vars["EXTRA"] = "x"; f.mu.Unlock() },
				Config:    varsConfig(`{ LOG_LEVEL = "info", REGION = "us" }`),
				ConfigPlanChecks: resource.ConfigPlanChecks{PreApply: []plancheck.PlanCheck{
					plancheck.ExpectResourceAction(addr, plancheck.ResourceActionUpdate),
				}},
				Check: f.check(map[string]string{"LOG_LEVEL": "info", "REGION": "us"}),
			},
			{
				Config: varsConfig(`{ LOG_LEVEL = "debug" }`),
				Check:  f.check(map[string]string{"LOG_LEVEL": "debug"}),
			},
			{
				Config: varsConfig(`{}`),
				Check:  f.check(map[string]string{}),
			},
			{
				Config: varsConfig(`{ A = "1" }`),
			},
			{
				ResourceName:      addr,
				ImportState:       true,
				ImportStateId:     fakeEnvID,
				ImportStateVerify: true,
			},
		},
	})
}
