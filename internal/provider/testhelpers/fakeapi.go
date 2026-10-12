package testhelpers

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/providerserver"
	"github.com/hashicorp/terraform-plugin-go/tfprotov6"

	"github.com/computesphere/terraform-provider-computesphere/internal/provider"
	"github.com/computesphere/terraform-provider-computesphere/internal/provider/spherelet"
)

// FakeAPI is an in-memory stand-in for the parts of the v2 API that carry
// spherelet types: the type list, services and deployments. Tests use it for
// API behaviour no cassette has been recorded for yet.
type FakeAPI struct {
	mu       sync.Mutex
	seq      int
	services map[string]map[string]any
	deps     map[string]map[string]any
	// Requests logs "METHOD /path?query body" for every request, in order.
	Requests []string
}

type fakeSphereletType struct {
	ID                string   `json:"id"`
	Object            string   `json:"object"`
	Name              string   `json:"name"`
	Family            string   `json:"family"`
	VCPU              float64  `json:"vcpu"`
	CPUGuaranteedVCPU float64  `json:"cpu_guaranteed_vcpu"`
	MemoryMB          int      `json:"memory_mb"`
	MonthlyPriceUSD   int      `json:"monthly_price_usd"`
	AnnualPriceUSD    int      `json:"annual_price_usd"`
	Available         bool     `json:"available"`
	Aliases           []string `json:"aliases"`
	planID            string
}

// FakePlanIDs are the plan ids the fake API reports for its types.
var FakePlanIDs = map[string]string{
	"sl-0.25c-512m": "00000000-0000-4000-8000-0000000000a1",
	"sl-1c-2g":      "00000000-0000-4000-8000-0000000000a2",
	"sl-2c-4g":      "00000000-0000-4000-8000-0000000000a3",
}

// fakeCatalog lists the fake API's spherelet types. Prices are test values.
var fakeCatalog = []fakeSphereletType{
	{"sl-0.25c-512m", "spherelet_type", "Flex 512 MB", "flex", 0.25, 0.25, 512, 10, 100, true, []string{"FLX"}, FakePlanIDs["sl-0.25c-512m"]},
	{"sl-1c-2g", "spherelet_type", "Standard 1", "standard", 1, 1, 2048, 20, 200, true, []string{"MAX"}, FakePlanIDs["sl-1c-2g"]},
	{"sl-2c-4g", "spherelet_type", "Standard 2", "standard", 2, 2, 4096, 30, 300, true, []string{"PWR"}, FakePlanIDs["sl-2c-4g"]},
}

// SetupFakeProvider starts a FakeAPI and returns provider factories pointed
// at it.
func SetupFakeProvider(t *testing.T) (map[string]func() (tfprotov6.ProviderServer, error), *FakeAPI) {
	t.Helper()
	f := &FakeAPI{services: map[string]map[string]any{}, deps: map[string]map[string]any{}}
	srv := httptest.NewServer(f.handler())
	t.Cleanup(srv.Close)
	return map[string]func() (tfprotov6.ProviderServer, error){
		"computesphere": providerserver.NewProtocol6WithError(provider.New("test",
			provider.WithHost(srv.URL+"/v2"),
			provider.WithAPIToken("some-api-token"),
			provider.WithAccountID(testAccountID),
			provider.WithHTTPClient(srv.Client()),
		)()),
	}, f
}

// Logged reports how many logged requests start with prefix.
func (f *FakeAPI) Logged(prefix string) int {
	f.mu.Lock()
	defer f.mu.Unlock()
	n := 0
	for _, r := range f.Requests {
		if strings.HasPrefix(r, prefix) {
			n++
		}
	}
	return n
}

func resolveFakeType(v string) *fakeSphereletType {
	for i := range fakeCatalog {
		it := &fakeCatalog[i]
		if it.ID == v || it.planID == v {
			return it
		}
		for _, a := range it.Aliases {
			if strings.EqualFold(a, v) {
				return it
			}
		}
	}
	return nil
}

// pickType resolves spherelet_type and/or plan_id from a write body with the
// API's rules: unknown is 400, and the two naming different types is 400.
func pickType(body map[string]any) (*fakeSphereletType, string) {
	st, _ := body[spherelet.FieldType].(string)
	pid, _ := body[spherelet.FieldPlanID].(string)
	var a, b *fakeSphereletType
	if st != "" {
		if a = resolveFakeType(st); a == nil {
			return nil, "unknown spherelet_type"
		}
	}
	if pid != "" {
		if b = resolveFakeType(pid); b == nil {
			return nil, "unknown plan_id"
		}
	}
	if a != nil && b != nil && a != b {
		return nil, "plan_id and spherelet_type name different types"
	}
	if a != nil {
		return a, ""
	}
	return b, ""
}

func setFakeType(o map[string]any, it *fakeSphereletType) {
	o[spherelet.FieldType] = it.ID
	o[spherelet.FieldPlanID] = it.planID
	o["spherelet_type_detail"] = map[string]any{"name": it.Name, "vcpu": it.VCPU, "memory_mb": it.MemoryMB}
}

func writeJSON(w http.ResponseWriter, code int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(v)
}

func writeProblem(w http.ResponseWriter, code int, detail string) {
	w.Header().Set("Content-Type", "application/problem+json")
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(map[string]any{"status": code, "title": http.StatusText(code), "detail": detail})
}

// deploymentView is a deployment read: the type sits in the nested record.
func deploymentView(d map[string]any) map[string]any {
	return map[string]any{
		"object": "deployment", "id": d["id"], "type": d["type"], "name": d["name"], "status": "Running",
		"raw": map[string]any{
			spherelet.FieldType:     d[spherelet.FieldType],
			"spherelet_type_detail": d["spherelet_type_detail"],
		},
	}
}

func (f *FakeAPI) handler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /v2/"+spherelet.ListPath, func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusOK, map[string]any{
			"object": "list", "items": fakeCatalog, "warnings": []any{},
			"pagination": map[string]any{"page": 1, "page_size": len(fakeCatalog), "total": len(fakeCatalog), "total_pages": 1},
		})
	})
	mux.HandleFunc("POST /v2/services", func(w http.ResponseWriter, r *http.Request) {
		body := decode(r)
		it, msg := pickType(body)
		if msg != "" {
			writeProblem(w, http.StatusBadRequest, msg)
			return
		}
		if it == nil {
			it = &fakeCatalog[0]
		}
		f.mu.Lock()
		defer f.mu.Unlock()
		f.seq++
		id := fmt.Sprintf("aaaaaaaa-0000-4000-8000-%012d", f.seq)
		s := map[string]any{
			"object": "service", "id": id, "name": body["name"], "type": body["type"], "project_id": body["project_id"],
			"plan": nil, "active": true, "last_opened_at": nil,
			"created_at": "2026-01-01T00:00:00Z", "updated_at": "2026-01-01T00:00:00Z",
		}
		setFakeType(s, it)
		f.services[id] = s
		writeJSON(w, http.StatusCreated, s)
	})
	mux.HandleFunc("GET /v2/services/{id}", func(w http.ResponseWriter, r *http.Request) {
		f.mu.Lock()
		defer f.mu.Unlock()
		s, ok := f.services[r.PathValue("id")]
		if !ok {
			writeProblem(w, http.StatusNotFound, "not found")
			return
		}
		writeJSON(w, http.StatusOK, s)
	})
	mux.HandleFunc("PATCH /v2/services/{id}", func(w http.ResponseWriter, r *http.Request) {
		body := decode(r)
		f.mu.Lock()
		defer f.mu.Unlock()
		s, ok := f.services[r.PathValue("id")]
		if !ok {
			writeProblem(w, http.StatusNotFound, "not found")
			return
		}
		it, msg := pickType(body)
		if msg != "" {
			writeProblem(w, http.StatusBadRequest, msg)
			return
		}
		if it != nil {
			setFakeType(s, it)
		}
		writeJSON(w, http.StatusOK, s)
	})
	mux.HandleFunc("DELETE /v2/services/{id}", func(w http.ResponseWriter, r *http.Request) {
		f.mu.Lock()
		delete(f.services, r.PathValue("id"))
		f.mu.Unlock()
		w.WriteHeader(http.StatusNoContent)
	})
	mux.HandleFunc("POST /v2/deployments", func(w http.ResponseWriter, r *http.Request) {
		body := decode(r)
		it, msg := pickType(body)
		if msg != "" {
			writeProblem(w, http.StatusBadRequest, msg)
			return
		}
		f.mu.Lock()
		defer f.mu.Unlock()
		svc := f.services[fmt.Sprint(body["service_id"])]
		if it == nil && svc != nil {
			it = resolveFakeType(fmt.Sprint(svc[spherelet.FieldType]))
		}
		if it == nil {
			it = &fakeCatalog[0]
		}
		f.seq++
		id := fmt.Sprintf("bbbbbbbb-0000-4000-8000-%012d", f.seq)
		name := "deployment"
		if svc != nil {
			name = fmt.Sprint(svc["name"])
		}
		d := map[string]any{"id": id, "type": body["type"], "name": name}
		setFakeType(d, it)
		f.deps[id] = d
		writeJSON(w, http.StatusCreated, deploymentView(d))
	})
	mux.HandleFunc("GET /v2/deployments/{id}", func(w http.ResponseWriter, r *http.Request) {
		f.mu.Lock()
		defer f.mu.Unlock()
		d, ok := f.deps[r.PathValue("id")]
		if !ok {
			writeProblem(w, http.StatusNotFound, "not found")
			return
		}
		writeJSON(w, http.StatusOK, deploymentView(d))
	})
	mux.HandleFunc("PATCH /v2/deployments/{id}", func(w http.ResponseWriter, r *http.Request) {
		body := decode(r)
		f.mu.Lock()
		defer f.mu.Unlock()
		d, ok := f.deps[r.PathValue("id")]
		if !ok {
			writeProblem(w, http.StatusNotFound, "not found")
			return
		}
		it, msg := pickType(body)
		if msg != "" {
			writeProblem(w, http.StatusBadRequest, msg)
			return
		}
		if it != nil {
			setFakeType(d, it)
		}
		writeJSON(w, http.StatusOK, deploymentView(d))
	})
	mux.HandleFunc("GET /v2/deployments/{id}/status", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusOK, map[string]any{
			"object": "deployment_status", "id": r.PathValue("id"), "status": "Running", "sphere_count": 1,
			"max_sphere_count": 0, "last_deployed_at": nil, "status_updated_at": "2026-01-01T00:00:00Z",
		})
	})
	mux.HandleFunc("POST /v2/deployments/{id}/deploy", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusAccepted, map[string]any{"object": "deployment", "id": r.PathValue("id"), "status": "Redeploying"})
	})
	mux.HandleFunc("DELETE /v2/deployments/{id}", func(w http.ResponseWriter, r *http.Request) {
		f.mu.Lock()
		delete(f.deps, r.PathValue("id"))
		f.mu.Unlock()
		w.WriteHeader(http.StatusNoContent)
	})
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		writeProblem(w, http.StatusNotFound, "no such route in the fake API: "+r.Method+" "+r.URL.Path)
	})
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body := ""
		if r.Body != nil {
			var raw json.RawMessage
			if err := json.NewDecoder(r.Body).Decode(&raw); err == nil {
				body = string(raw)
			}
			r.Body = jsonBody(raw)
		}
		entry := r.Method + " " + r.URL.Path
		if r.URL.RawQuery != "" {
			entry += "?" + r.URL.RawQuery
		}
		if body != "" {
			entry += " " + body
		}
		f.mu.Lock()
		f.Requests = append(f.Requests, entry)
		f.mu.Unlock()
		mux.ServeHTTP(w, r)
	})
}

func decode(r *http.Request) map[string]any {
	var m map[string]any
	_ = json.NewDecoder(r.Body).Decode(&m)
	if m == nil {
		m = map[string]any{}
	}
	return m
}

func jsonBody(raw []byte) io.ReadCloser {
	return io.NopCloser(bytes.NewReader(raw))
}
