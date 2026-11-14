package provider_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/providerserver"
	"github.com/hashicorp/terraform-plugin-go/tfprotov6"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/plancheck"

	"github.com/computesphere/terraform-provider-computesphere/internal/provider"
)

// fakeProjectAPI stores one project the way the API does: an empty
// description is left out of the response, not returned as "".
type fakeProjectAPI struct {
	mu      sync.Mutex
	desc    *string
	deleted bool
	patches []map[string]any
}

func (f *fakeProjectAPI) view() map[string]any {
	p := map[string]any{
		"object": "project", "id": "13316e7e-21d7-43c8-9ee7-69674510ceeb", "name": "example",
		"account_id": "11111111-1111-1111-1111-111111111111", "created_at": "2026-11-14T00:00:00Z",
	}
	if f.desc != nil && *f.desc != "" {
		p["description"] = *f.desc
	}
	return p
}

func (f *fakeProjectAPI) handler() http.Handler {
	write := func(w http.ResponseWriter, code int, v any) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(code)
		_ = json.NewEncoder(w).Encode(v)
	}
	mux := http.NewServeMux()
	mux.HandleFunc("POST /v2/projects", func(w http.ResponseWriter, r *http.Request) {
		var body struct {
			Description *string `json:"description"`
		}
		_ = json.NewDecoder(r.Body).Decode(&body)
		f.mu.Lock()
		defer f.mu.Unlock()
		f.desc = body.Description
		write(w, http.StatusCreated, f.view())
	})
	mux.HandleFunc("GET /v2/projects/{id}", func(w http.ResponseWriter, r *http.Request) {
		f.mu.Lock()
		defer f.mu.Unlock()
		if f.deleted {
			write(w, http.StatusNotFound, map[string]any{"status": 404, "title": "Not Found"})
			return
		}
		write(w, http.StatusOK, f.view())
	})
	mux.HandleFunc("PATCH /v2/projects/{id}", func(w http.ResponseWriter, r *http.Request) {
		var body map[string]any
		_ = json.NewDecoder(r.Body).Decode(&body)
		f.mu.Lock()
		defer f.mu.Unlock()
		f.patches = append(f.patches, body)
		if d, ok := body["description"].(string); ok {
			f.desc = &d
		}
		write(w, http.StatusOK, f.view())
	})
	mux.HandleFunc("DELETE /v2/projects/{id}", func(w http.ResponseWriter, r *http.Request) {
		f.mu.Lock()
		f.deleted = true
		f.mu.Unlock()
		w.WriteHeader(http.StatusNoContent)
	})
	return mux
}

func setupFakeProjectAPI(t *testing.T) (map[string]func() (tfprotov6.ProviderServer, error), *fakeProjectAPI) {
	t.Helper()
	f := &fakeProjectAPI{}
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

// A project without a description applies and re-plans clean although the
// API leaves the field out; adding one and removing it again both apply, and
// the removal sends "" so the API clears it.
func TestAccProjectResource_noDescription(t *testing.T) {
	factories, f := setupFakeProjectAPI(t)
	const addr = "computesphere_project.example"
	bare := `resource "computesphere_project" "example" { name = "example" }`
	withDesc := `resource "computesphere_project" "example" {
  name        = "example"
  description = "hello"
}`
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: factories,
		Steps: []resource.TestStep{
			{
				Config: bare,
				Check:  resource.TestCheckResourceAttr(addr, "description", ""),
			},
			{
				Config:           bare,
				ConfigPlanChecks: resource.ConfigPlanChecks{PreApply: []plancheck.PlanCheck{plancheck.ExpectEmptyPlan()}},
			},
			{
				Config: withDesc,
				Check:  resource.TestCheckResourceAttr(addr, "description", "hello"),
			},
			{
				Config: bare,
				Check:  resource.TestCheckResourceAttr(addr, "description", ""),
			},
		},
	})
	f.mu.Lock()
	defer f.mu.Unlock()
	if n := len(f.patches); n != 2 {
		t.Fatalf("%d PATCH requests, want 2", n)
	}
	if d, ok := f.patches[1]["description"].(string); !ok || d != "" {
		t.Fatalf("removing the description sent %v, want \"\"", f.patches[1]["description"])
	}
}
