package provider_test

import (
	"encoding/json"
	"fmt"
	"net/http"
	"regexp"
	"strings"
	"sync"
	"testing"
	"time"

	dbresource "github.com/computesphere/terraform-provider-computesphere/internal/provider/database/resource"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/plancheck"

	th "github.com/computesphere/terraform-provider-computesphere/internal/provider/testhelpers"
)

// fakeDatabasesAPI serves the /v2/databases routes from memory; a database
// is created on version 17 and is available at once.
type fakeDatabasesAPI struct {
	mu      sync.Mutex
	db      map[string]any
	creates int
}

func (f *fakeDatabasesAPI) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	f.mu.Lock()
	defer f.mu.Unlock()
	w.Header().Set("Content-Type", "application/json")
	const id = "0c1d2e3f-4a5b-4c6d-8e7f-001122334455"
	switch {
	case r.Method == http.MethodPost && r.URL.Path == "/v2/databases":
		var body map[string]any
		_ = json.NewDecoder(r.Body).Decode(&body)
		f.creates++
		f.db = map[string]any{
			"object": "database", "id": id, "name": body["name"], "engine": "postgresql", "version": "17",
			"plan": "standard", "instance_type": body["instance_type"], "high_availability": "none",
			"status": "available", "redundancy": "single-zone", "storage_gb": 10, "members": []any{},
			"account_id": "11111111-1111-1111-1111-111111111111", "project_id": "22222222-2222-2222-2222-222222222222",
			"environment_id": body["environment_id"], "region": "eastus2", "error": "",
			"created_at": "2026-10-12T10:00:00Z", "updated_at": "2026-10-12T10:00:00Z",
			"links": map[string]any{"self": "/v2/databases/" + id},
		}
		w.WriteHeader(http.StatusCreated)
		_ = json.NewEncoder(w).Encode(f.db)
	case strings.HasPrefix(r.URL.Path, "/v2/databases/"+id) && f.db != nil:
		if r.Method == http.MethodDelete {
			f.db = nil
			w.WriteHeader(http.StatusAccepted)
			return
		}
		_ = json.NewEncoder(w).Encode(f.db)
	default:
		w.WriteHeader(http.StatusNotFound)
	}
}

// An engine or version change on an existing database is refused at plan
// time, never planned as a replacement that would start an empty database.
func TestDatabaseResource_versionChangeRefused(t *testing.T) {
	dbresource.SetPollIntervalForTest(time.Millisecond)
	api := &fakeDatabasesAPI{}
	base := `
resource "computesphere_database" "example" {
  name           = "app-db"
  environment_id = "33333333-3333-3333-3333-333333333333"
  instance_type  = "pg-1c-2g"
  inject         = false
%s
}
`
	name := "computesphere_database.example"
	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: th.SetupFakeAPIProvider(t, api),
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(base, ""),
				Check:  resource.TestCheckResourceAttr(name, "version", "17"),
			},
			{
				Config:      fmt.Sprintf(base, `  version = "18"`),
				ExpectError: regexp.MustCompile(`version can't change on an existing database`),
			},
			{
				Config: fmt.Sprintf(base, `  version = "17"`),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{plancheck.ExpectResourceAction(name, plancheck.ResourceActionNoop)},
				},
			},
		},
	})
	if api.creates != 1 {
		t.Fatalf("want 1 create (no replacement), got %d", api.creates)
	}
}
