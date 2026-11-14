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

// fakeSecretsAPI keeps an environment's secrets like the API does: PUT
// replaces the set, a blank value keeps the stored one, reads return names
// with empty values.
type fakeSecretsAPI struct {
	mu      sync.Mutex
	secrets map[string]string
	last    map[string]string // the last PUT's secrets
}

func (f *fakeSecretsAPI) handler() http.Handler {
	write := func(w http.ResponseWriter, code int, v any) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(code)
		_ = json.NewEncoder(w).Encode(v)
	}
	names := func() map[string]string {
		out := map[string]string{}
		for k := range f.secrets {
			out[k] = ""
		}
		return out
	}
	mux := http.NewServeMux()
	mux.HandleFunc("GET /v2/environments/{eid}/secrets", func(w http.ResponseWriter, r *http.Request) {
		f.mu.Lock()
		defer f.mu.Unlock()
		write(w, http.StatusOK, map[string]any{"object": "environment_secrets", "secrets": names()})
	})
	mux.HandleFunc("PUT /v2/environments/{eid}/secrets", func(w http.ResponseWriter, r *http.Request) {
		var body struct {
			Secrets    map[string]string `json:"secrets"`
			AllowEmpty bool              `json:"allow_empty"`
		}
		_ = json.NewDecoder(r.Body).Decode(&body)
		f.mu.Lock()
		defer f.mu.Unlock()
		f.last = body.Secrets
		if len(body.Secrets) == 0 && !body.AllowEmpty {
			write(w, http.StatusUnprocessableEntity, map[string]any{"status": 422, "title": "allow_empty is required"})
			return
		}
		next := map[string]string{}
		for k, v := range body.Secrets {
			if v == "" {
				old, ok := f.secrets[k]
				if !ok {
					write(w, http.StatusBadRequest, map[string]any{"status": 400, "title": "a new secret needs a value"})
					return
				}
				v = old
			}
			next[k] = v
		}
		f.secrets = next
		write(w, http.StatusOK, map[string]any{"object": "environment_secrets", "secrets": names()})
	})
	return mux
}

func (f *fakeSecretsAPI) check(stored, sent map[string]string) resource.TestCheckFunc {
	return func(*terraform.State) error {
		f.mu.Lock()
		defer f.mu.Unlock()
		if fmt.Sprint(f.secrets) != fmt.Sprint(stored) {
			return fmt.Errorf("stored = %v, want %v", f.secrets, stored)
		}
		if sent != nil && fmt.Sprint(f.last) != fmt.Sprint(sent) {
			return fmt.Errorf("last PUT sent %v, want %v", f.last, sent)
		}
		return nil
	}
}

func setupFakeSecretsAPI(t *testing.T) (map[string]func() (tfprotov6.ProviderServer, error), *fakeSecretsAPI) {
	t.Helper()
	f := &fakeSecretsAPI{secrets: map[string]string{}}
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

func secretsConfig(body string) string {
	return `resource "computesphere_environment_secrets" "example" {
  environment_id = "` + fakeEnvID + `"
  secrets        = ` + body + `
}`
}

func TestAccEnvironmentSecretsResource(t *testing.T) {
	factories, f := setupFakeSecretsAPI(t)
	const addr = "computesphere_environment_secrets.example"
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: factories,
		CheckDestroy: func(*terraform.State) error {
			f.mu.Lock()
			defer f.mu.Unlock()
			if len(f.secrets) != 0 {
				return fmt.Errorf("secrets left after destroy: %v", f.secrets)
			}
			return nil
		},
		Steps: []resource.TestStep{
			{
				Config: secretsConfig(`{ API_KEY = "k1", DB_PASS = "p1" }`),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(addr, "secrets.API_KEY", "k1"),
					f.check(map[string]string{"API_KEY": "k1", "DB_PASS": "p1"}, nil),
				),
			},
			{
				// The API returns no values; the configured ones stay in state.
				Config:           secretsConfig(`{ API_KEY = "k1", DB_PASS = "p1" }`),
				ConfigPlanChecks: resource.ConfigPlanChecks{PreApply: []plancheck.PlanCheck{plancheck.ExpectEmptyPlan()}},
			},
			{
				// Only the changed value is sent; the unchanged one goes blank.
				Config: secretsConfig(`{ API_KEY = "k2", DB_PASS = "p1" }`),
				Check:  f.check(map[string]string{"API_KEY": "k2", "DB_PASS": "p1"}, map[string]string{"API_KEY": "k2", "DB_PASS": ""}),
			},
			{
				// A secret added elsewhere shows as a change that removes it.
				PreConfig: func() { f.mu.Lock(); f.secrets["EXTRA"] = "x"; f.mu.Unlock() },
				Config:    secretsConfig(`{ API_KEY = "k2", DB_PASS = "p1" }`),
				ConfigPlanChecks: resource.ConfigPlanChecks{PreApply: []plancheck.PlanCheck{
					plancheck.ExpectResourceAction(addr, plancheck.ResourceActionUpdate),
				}},
				Check: f.check(map[string]string{"API_KEY": "k2", "DB_PASS": "p1"}, nil),
			},
			{
				Config: secretsConfig(`{}`),
				Check:  f.check(map[string]string{}, map[string]string{}),
			},
			{
				Config: secretsConfig(`{ A = "1" }`),
			},
		},
	})
}
