package provider_test

import (
	"encoding/json"
	"net/http"
	"strings"
	"sync"
	"testing"
)

// fakeQueueAPI is an in-memory stand-in for the /v2/queues routes. Request
// bodies are decoded strictly, like the API's closed schemas, so a field the
// spec doesn't list fails the test. A read after a create or resize reports
// the transitional status once, then available.
type fakeQueueAPI struct {
	t       *testing.T
	mu      sync.Mutex
	queues  map[string]map[string]any
	pending map[string]bool
	calls   []string
}

func newFakeQueueAPI(t *testing.T) *fakeQueueAPI {
	return &fakeQueueAPI{t: t, queues: map[string]map[string]any{}, pending: map[string]bool{}}
}

const fakeQueueID = "0b5d6a8e-2c1f-4e7a-9d3b-6f0e1c2a3b4c"

func (f *fakeQueueAPI) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.calls = append(f.calls, r.Method+" "+r.URL.Path)
	if r.Header.Get("Authorization") != "Bearer some-api-token" || r.Header.Get("X-Account-Id") == "" {
		w.WriteHeader(http.StatusUnauthorized)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	p := strings.TrimPrefix(r.URL.Path, "/v2")
	switch {
	case r.Method == http.MethodPost && p == "/queues":
		var body struct {
			Name          string `json:"name"`
			EnvironmentID string `json:"environment_id"`
			Engine        string `json:"engine"`
			Version       string `json:"version"`
			InstanceType  string `json:"instance_type"`
		}
		if !f.decode(w, r, &body) {
			return
		}
		c := map[string]any{
			"object": "queue", "id": fakeQueueID, "name": body.Name, "engine": body.Engine, "version": "4.3",
			"instance_type": body.InstanceType, "nodes": nodes(body.InstanceType),
			"status": "provisioning", "read_only": false,
			"account_id": "11111111-1111-1111-1111-111111111111", "project_id": "22222222-2222-2222-2222-222222222222",
			"environment_id": body.EnvironmentID, "region": "eastus2", "error": "",
			"created_at": "2026-10-12T10:00:00Z", "updated_at": "2026-10-12T10:00:00Z",
			"links": map[string]any{"self": "/v2/queues/" + fakeQueueID},
		}
		f.queues[fakeQueueID] = c
		f.pending[fakeQueueID] = true
		w.WriteHeader(http.StatusCreated)
		_ = json.NewEncoder(w).Encode(c)
	case strings.HasPrefix(p, "/queues/"):
		id := strings.TrimPrefix(p, "/queues/")
		c, ok := f.queues[id]
		if !ok {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		switch r.Method {
		case http.MethodGet:
			out := map[string]any{}
			for k, v := range c {
				out[k] = v
			}
			if f.pending[id] {
				f.pending[id] = false
			} else {
				c["status"] = "available"
				out["status"] = "available"
			}
			_ = json.NewEncoder(w).Encode(out)
		case http.MethodPatch:
			var body struct {
				InstanceType *string `json:"instance_type"`
			}
			if !f.decode(w, r, &body) {
				return
			}
			if body.InstanceType != nil {
				c["instance_type"] = *body.InstanceType
				c["nodes"] = nodes(*body.InstanceType)
			}
			c["status"] = "resizing"
			f.pending[id] = true
			_ = json.NewEncoder(w).Encode(c)
		case http.MethodDelete:
			delete(f.queues, id)
			w.WriteHeader(http.StatusNoContent)
		default:
			w.WriteHeader(http.StatusMethodNotAllowed)
		}
	default:
		w.WriteHeader(http.StatusNotFound)
	}
}

// decode reads a closed request body, answering 400 on an unknown field.
func (f *fakeQueueAPI) decode(w http.ResponseWriter, r *http.Request, into any) bool {
	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()
	if err := dec.Decode(into); err != nil {
		f.t.Errorf("%s %s: request body doesn't match the closed schema: %v", r.Method, r.URL.Path, err)
		w.WriteHeader(http.StatusBadRequest)
		return false
	}
	return true
}

func (f *fakeQueueAPI) called(prefix string) bool {
	f.mu.Lock()
	defer f.mu.Unlock()
	for _, c := range f.calls {
		if strings.HasPrefix(c, prefix) {
			return true
		}
	}
	return false
}

func nodes(instanceType string) int {
	if strings.HasSuffix(instanceType, "-ha") {
		return 3
	}
	return 1
}
