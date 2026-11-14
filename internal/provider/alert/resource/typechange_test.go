package provider_test

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"regexp"
	"sync"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/providerserver"
	"github.com/hashicorp/terraform-plugin-go/tfprotov6"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"

	"github.com/computesphere/terraform-provider-computesphere/internal/provider"
)

const fakeAlertID = "66666666-6666-4666-8666-666666666666"

// fakeAlertAPI stores one rule. A job_failing_streak rule without a threshold
// gets 3, and a job rule's effective period is 0, as in the API.
type fakeAlertAPI struct {
	mu      sync.Mutex
	rule    map[string]any
	patches []map[string]any
}

func (f *fakeAlertAPI) handler() http.Handler {
	write := func(w http.ResponseWriter, code int, v any) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(code)
		_ = json.NewEncoder(w).Encode(v)
	}
	fill := func() {
		t, _ := f.rule["alert_type"].(string)
		if _, ok := f.rule["threshold"]; !ok {
			f.rule["threshold"] = 0
			if t == "job_failing_streak" {
				f.rule["threshold"] = 3
			}
		}
		if _, ok := f.rule["evaluation_period"]; !ok {
			f.rule["evaluation_period"] = 0
		}
		f.rule["effective_evaluation_period"] = 0
		if t == "cpu" || t == "memory" || t == "disk" {
			f.rule["effective_evaluation_period"] = 120
		}
	}
	mux := http.NewServeMux()
	mux.HandleFunc("POST /v2/alerts", func(w http.ResponseWriter, r *http.Request) {
		var b map[string]any
		_ = json.NewDecoder(r.Body).Decode(&b)
		f.mu.Lock()
		defer f.mu.Unlock()
		f.rule = b
		f.rule["object"], f.rule["id"], f.rule["active"], f.rule["status"] = "alert_rule", fakeAlertID, true, "active"
		f.rule["project_alert_enabled"], f.rule["created_at"] = true, "2026-11-14T00:00:00Z"
		fill()
		write(w, http.StatusCreated, f.rule)
	})
	mux.HandleFunc("GET /v2/alerts/{id}", func(w http.ResponseWriter, r *http.Request) {
		f.mu.Lock()
		defer f.mu.Unlock()
		if f.rule == nil {
			write(w, http.StatusNotFound, map[string]any{"status": 404})
			return
		}
		write(w, http.StatusOK, f.rule)
	})
	mux.HandleFunc("PATCH /v2/alerts/{id}", func(w http.ResponseWriter, r *http.Request) {
		var b map[string]any
		_ = json.NewDecoder(r.Body).Decode(&b)
		f.mu.Lock()
		defer f.mu.Unlock()
		f.patches = append(f.patches, b)
		if b["alert_type"] != f.rule["alert_type"] {
			delete(f.rule, "threshold")
			delete(f.rule, "evaluation_period")
		}
		for k, v := range b {
			f.rule[k] = v
		}
		fill()
		write(w, http.StatusOK, f.rule)
	})
	mux.HandleFunc("DELETE /v2/alerts/{id}", func(w http.ResponseWriter, r *http.Request) {
		f.mu.Lock()
		f.rule = nil
		f.mu.Unlock()
		w.WriteHeader(http.StatusNoContent)
	})
	return mux
}

func setupFakeAlertAPI(t *testing.T) (map[string]func() (tfprotov6.ProviderServer, error), *fakeAlertAPI) {
	t.Helper()
	f := &fakeAlertAPI{}
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

func alertConfig(body string) string {
	return `resource "computesphere_alert" "example" {
  project_id     = "13316e7e-21d7-43c8-9ee7-69674510ceeb"
  environment_id = "77777777-7777-4777-8777-777777777777"
  severity       = "high"
` + body + `
}`
}

// Changing alert_type doesn't carry the old threshold over: the PATCH leaves
// it out and the new rule's own value is read back. Thresholds outside the
// type's range are refused at plan time.
func TestAccAlertResource_typeChange(t *testing.T) {
	factories, f := setupFakeAlertAPI(t)
	const addr = "computesphere_alert.example"
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: factories,
		Steps: []resource.TestStep{
			{
				Config: alertConfig(`  alert_type        = "cpu"
  threshold         = 80
  evaluation_period = 300`),
				Check: resource.TestCheckResourceAttr(addr, "effective_evaluation_period", "120"),
			},
			{
				Config: alertConfig(`  alert_type = "job_failing_streak"`),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(addr, "threshold", "3"),
					resource.TestCheckResourceAttr(addr, "effective_evaluation_period", "0"),
					func(*terraform.State) error {
						f.mu.Lock()
						defer f.mu.Unlock()
						last := f.patches[len(f.patches)-1]
						if _, ok := last["threshold"]; ok {
							return fmt.Errorf("threshold carried over in %v", last)
						}
						return nil
					},
				),
			},
			{
				Config:      alertConfig(`  alert_type = "memory"`),
				ExpectError: regexp.MustCompile(`needs threshold`),
			},
			{
				Config: alertConfig(`  alert_type = "disk"
  threshold  = 90`),
				ExpectError: regexp.MustCompile(`needs evaluation_period`),
			},
			{
				Config: alertConfig(`  alert_type = "job_failing_streak"
  threshold  = 20`),
				ExpectError: regexp.MustCompile(`from 2 to 10`),
			},
			{
				Config: alertConfig(`  alert_type = "job_failing_streak"`),
			},
		},
	})
}
