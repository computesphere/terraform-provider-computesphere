package provider_test

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"regexp"
	"strings"
	"sync"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/providerserver"
	"github.com/hashicorp/terraform-plugin-go/tfprotov6"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"

	"github.com/computesphere/terraform-provider-computesphere/internal/provider"
)

const fakeSourceID = "88888888-8888-4888-8888-888888888888"

// fakeSourceAPI keeps one secret manager. It records every write; a read
// never returns the credential, and all_projects lists every project.
type fakeSourceAPI struct {
	mu     sync.Mutex
	source map[string]any
	writes []string // "METHOD path body"
}

func (f *fakeSourceAPI) handler() http.Handler {
	write := func(w http.ResponseWriter, code int, v any) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(code)
		_ = json.NewEncoder(w).Encode(v)
	}
	view := func() map[string]any {
		out := map[string]any{}
		for k, v := range f.source {
			if k != "credential" {
				out[k] = v
			}
		}
		return out
	}
	mux := http.NewServeMux()
	mux.HandleFunc("POST /v2/secret-sources", func(w http.ResponseWriter, r *http.Request) {
		var b map[string]any
		_ = json.NewDecoder(r.Body).Decode(&b)
		f.source = b
		f.source["object"], f.source["id"], f.source["status"] = "secret_source", fakeSourceID, "ok"
		f.source["credential_fingerprint"], f.source["ref_count"], f.source["sync_interval_s"] = "fp1", 0, 300
		write(w, http.StatusCreated, view())
	})
	mux.HandleFunc("GET /v2/secret-sources/{sid}", func(w http.ResponseWriter, r *http.Request) {
		if f.source == nil {
			write(w, http.StatusNotFound, map[string]any{"status": 404})
			return
		}
		write(w, http.StatusOK, view())
	})
	mux.HandleFunc("PATCH /v2/secret-sources/{sid}", func(w http.ResponseWriter, r *http.Request) {
		var b map[string]any
		_ = json.NewDecoder(r.Body).Decode(&b)
		for k, v := range b {
			f.source[k] = v
		}
		if _, ok := b["credential"]; ok {
			f.source["credential_fingerprint"] = "fp2"
		}
		write(w, http.StatusOK, view())
	})
	mux.HandleFunc("PUT /v2/secret-sources/{sid}/access", func(w http.ResponseWriter, r *http.Request) {
		var b map[string]any
		_ = json.NewDecoder(r.Body).Decode(&b)
		if b["all_projects"] == true {
			b["projects"] = []map[string]any{{"project_id": "p-1", "all_environments": true}, {"project_id": "p-2", "all_environments": true}}
		}
		f.source["access"] = b
		write(w, http.StatusOK, view())
	})
	mux.HandleFunc("DELETE /v2/secret-sources/{sid}", func(w http.ResponseWriter, r *http.Request) {
		f.source = nil
		w.WriteHeader(http.StatusNoContent)
	})
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		f.mu.Lock()
		defer f.mu.Unlock()
		if r.Method != http.MethodGet {
			var raw json.RawMessage
			_ = json.NewDecoder(r.Body).Decode(&raw)
			f.writes = append(f.writes, strings.TrimSpace(r.Method+" "+r.URL.Path+" "+string(raw)))
			r.Body = http.NoBody
			if len(raw) > 0 {
				r.Body = nopBody(string(raw))
			}
		}
		mux.ServeHTTP(w, r)
	})
}

func (f *fakeSourceAPI) last(prefix string) string {
	f.mu.Lock()
	defer f.mu.Unlock()
	for i := len(f.writes) - 1; i >= 0; i-- {
		if strings.HasPrefix(f.writes[i], prefix) {
			return f.writes[i]
		}
	}
	return ""
}

func setupFakeSourceAPI(t *testing.T) (map[string]func() (tfprotov6.ProviderServer, error), *fakeSourceAPI) {
	t.Helper()
	f := &fakeSourceAPI{}
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

func sourceConfig(name string, version int, extra string) string {
	return fmt.Sprintf(`resource "computesphere_secret_source" "example" {
  name               = %q
  type               = "doppler"
  auth_type          = "service_token"
  config             = { project = "web", config = "prd" }
  credential         = { token = "dp.st.example" }
  credential_version = %d
%s
}`, name, version, extra)
}

func TestAccSecretSourceResource(t *testing.T) {
	factories, f := setupFakeSourceAPI(t)
	const addr = "computesphere_secret_source.example"
	noCredentialInState := func(s *terraform.State) error {
		if v, ok := s.RootModule().Resources[addr].Primary.Attributes["credential.token"]; ok {
			return fmt.Errorf("credential in state: %q", v)
		}
		return nil
	}
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: factories,
		Steps: []resource.TestStep{
			{
				Config: sourceConfig("doppler-web", 1, ""),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(addr, "status", "ok"),
					resource.TestCheckResourceAttr(addr, "all_projects", "false"),
					noCredentialInState,
					func(*terraform.State) error {
						if w := f.last("POST /v2/secret-sources"); !strings.Contains(w, `"credential":{"token":"dp.st.example"}`) {
							return fmt.Errorf("create didn't send the credential: %s", w)
						}
						return nil
					},
				),
			},
			{
				// A rename doesn't send the credential again.
				Config: sourceConfig("doppler-web-2", 1, ""),
				Check: func(*terraform.State) error {
					if w := f.last("PATCH "); strings.Contains(w, "credential") || !strings.Contains(w, "doppler-web-2") {
						return fmt.Errorf("rename PATCH = %s", w)
					}
					return nil
				},
			},
			{
				// A new credential_version does.
				Config: sourceConfig("doppler-web-2", 2, ""),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(addr, "credential_fingerprint", "fp2"),
					noCredentialInState,
					func(*terraform.State) error {
						if w := f.last("PATCH "); !strings.Contains(w, `"credential":{"token":"dp.st.example"}`) {
							return fmt.Errorf("rotation PATCH = %s", w)
						}
						return nil
					},
				),
			},
			{
				// A listed project is allowed all its environments.
				Config: sourceConfig("doppler-web-2", 2, `  project_ids = ["p-1"]`),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(addr, "project_ids.#", "1"),
					func(*terraform.State) error {
						if w := f.last("PUT "); !strings.Contains(w, `{"all_environments":true,"project_id":"p-1"}`) {
							return fmt.Errorf("access PUT = %s", w)
						}
						return nil
					},
				),
			},
			{
				Config: sourceConfig("doppler-web-2", 2, `  all_projects = true`),
				Check:  resource.TestCheckResourceAttr(addr, "project_ids.#", "0"),
			},
			{
				Config:      sourceConfig("doppler-web-2", 2, "  all_projects = true\n  project_ids = [\"p-1\"]"),
				ExpectError: regexp.MustCompile(`not both`),
			},
			{
				Config: sourceConfig("doppler-web-2", 2, ""),
			},
		},
	})
}

func nopBody(s string) *readCloser { return &readCloser{strings.NewReader(s)} }

type readCloser struct{ *strings.Reader }

func (readCloser) Close() error { return nil }
