package provider_test

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"regexp"
	"strings"
	"sync"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/providerserver"
	"github.com/hashicorp/terraform-plugin-go/tfprotov6"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/plancheck"
	"github.com/hashicorp/terraform-plugin-testing/terraform"

	"github.com/computesphere/terraform-provider-computesphere/internal/provider"
)

const (
	envAddr     = "computesphere_environment.example"
	fakeEnvID   = "77777777-7777-4777-8777-777777777777"
	projectA    = "13316e7e-21d7-43c8-9ee7-69674510ceeb"
	projectB    = "24427f8f-32e8-44d9-8ab7-7a785621dffc"
	envRegion   = "us-east-1"
	otherRegion = "eu-north-1"
)

// fakeEnvAPI is one environment: create, read, rename, move and delete.
type fakeEnvAPI struct {
	mu       sync.Mutex
	env      map[string]any
	services int // deployments the environment runs
	deleted  bool
	sent     []string // "METHOD /path body", writes only
}

func (f *fakeEnvAPI) handler() http.Handler {
	write := func(w http.ResponseWriter, code int, v any) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(code)
		_ = json.NewEncoder(w).Encode(v)
	}
	mux := http.NewServeMux()
	mux.HandleFunc("POST /v2/environments", func(w http.ResponseWriter, r *http.Request) {
		var b map[string]any
		_ = json.NewDecoder(r.Body).Decode(&b)
		f.env = map[string]any{
			"object": "environment", "id": fakeEnvID, "name": b["name"], "region": b["region"],
			"project_id": b["project_id"], "created_at": "2026-11-14T00:00:00Z", "updated_at": "2026-11-14T00:00:00Z",
		}
		write(w, http.StatusCreated, f.env)
	})
	mux.HandleFunc("GET /v2/environments/{eid}", func(w http.ResponseWriter, r *http.Request) {
		if f.deleted || f.env == nil {
			write(w, http.StatusNotFound, map[string]any{"status": 404, "title": "Not Found"})
			return
		}
		write(w, http.StatusOK, f.env)
	})
	mux.HandleFunc("PATCH /v2/environments/{eid}", func(w http.ResponseWriter, r *http.Request) {
		var b map[string]any
		_ = json.NewDecoder(r.Body).Decode(&b)
		if n, ok := b["name"]; ok {
			f.env["name"] = n
		}
		write(w, http.StatusOK, f.env)
	})
	mux.HandleFunc("POST /v2/environments/{eid}/move", func(w http.ResponseWriter, r *http.Request) {
		var b map[string]any
		_ = json.NewDecoder(r.Body).Decode(&b)
		f.env["project_id"] = b["target_project_id"]
		write(w, http.StatusOK, f.env)
	})
	mux.HandleFunc("GET /v2/deployments", func(w http.ResponseWriter, r *http.Request) {
		items := []map[string]any{}
		for i := 0; i < f.services; i++ {
			items = append(items, map[string]any{"object": "deployment", "id": fmt.Sprintf("dep-%d", i)})
		}
		write(w, http.StatusOK, map[string]any{"object": "list", "items": items,
			"pagination": map[string]any{"page": 1, "page_size": 1, "total": f.services, "total_pages": 1}})
	})
	mux.HandleFunc("DELETE /v2/environments/{eid}", func(w http.ResponseWriter, r *http.Request) {
		f.deleted = true
		w.WriteHeader(http.StatusNoContent)
	})
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		f.mu.Lock()
		defer f.mu.Unlock()
		if r.Method != http.MethodGet {
			var raw json.RawMessage
			_ = json.NewDecoder(r.Body).Decode(&raw)
			f.sent = append(f.sent, strings.TrimSpace(r.Method+" "+r.URL.Path+" "+string(raw)))
			r.Body = http.NoBody
			if len(raw) > 0 {
				r.Body = io.NopCloser(strings.NewReader(string(raw)))
			}
		}
		mux.ServeHTTP(w, r)
	})
}

func (f *fakeEnvAPI) reset() {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.sent = nil
}

func (f *fakeEnvAPI) expect(want ...string) resource.TestCheckFunc {
	return func(*terraform.State) error {
		f.mu.Lock()
		defer f.mu.Unlock()
		if fmt.Sprint(f.sent) != fmt.Sprint(want) {
			return fmt.Errorf("sent %v, want %v", f.sent, want)
		}
		return nil
	}
}

func setupFakeEnvAPI(t *testing.T) (map[string]func() (tfprotov6.ProviderServer, error), *fakeEnvAPI) {
	t.Helper()
	f := &fakeEnvAPI{}
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

func envConfig(project, region string) string {
	return fmt.Sprintf(`resource "computesphere_environment" "example" {
  name       = "production"
  region     = %q
  project_id = %q
}`, region, project)
}

// A project change on an environment that runs services is refused at plan
// time (the move would take them along); without services it moves the
// environment in place. A region change is refused at plan time. Refusals
// send nothing.
func TestAccEnvironmentResource_moveAndRegion(t *testing.T) {
	factories, f := setupFakeEnvAPI(t)
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: factories,
		Steps: []resource.TestStep{
			{Config: envConfig(projectA, envRegion)},
			{
				PreConfig:   func() { f.reset(); f.mu.Lock(); f.services = 2; f.mu.Unlock() },
				Config:      envConfig(projectB, envRegion),
				ExpectError: regexp.MustCompile(`project_id can't change while the environment runs services`),
			},
			{
				PreConfig:        func() { f.reset(); f.mu.Lock(); f.services = 0; f.mu.Unlock() },
				Config:           envConfig(projectB, envRegion),
				ConfigPlanChecks: resource.ConfigPlanChecks{PreApply: []plancheck.PlanCheck{plancheck.ExpectResourceAction(envAddr, plancheck.ResourceActionUpdate)}},
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(envAddr, "project_id", projectB),
					resource.TestCheckResourceAttr(envAddr, "id", fakeEnvID),
					f.expect(
						`POST /v2/environments/`+fakeEnvID+`/move {"target_project_id":"`+projectB+`"}`,
						`PATCH /v2/environments/`+fakeEnvID+` {"name":"production"}`,
					),
				),
			},
			{
				PreConfig:   f.reset,
				Config:      envConfig(projectB, otherRegion),
				ExpectError: regexp.MustCompile(`region can't change on an existing environment`),
			},
			{
				Config: envConfig(projectB, envRegion),
				Check:  f.expect(),
			},
		},
	})
}
