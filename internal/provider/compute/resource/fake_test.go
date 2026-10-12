package provider_test

import (
	"encoding/json"
	"fmt"
	"net/http"
	"sort"
	"strings"
	"sync"
	"testing"
)

// fakeDeploymentsAPI is an in-memory stand-in for the /v2/deployments
// routes the compute resources call. It records each request body so tests
// can check it against the type's closed create and change bodies.
type fakeDeploymentsAPI struct {
	t     *testing.T
	mu    sync.Mutex
	deps  map[string]map[string]any
	seq   int
	calls []call
}

type call struct {
	method, path string
	body         map[string]any
}

func newFakeDeploymentsAPI(t *testing.T) *fakeDeploymentsAPI {
	return &fakeDeploymentsAPI{t: t, deps: map[string]map[string]any{}}
}

func (f *fakeDeploymentsAPI) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	f.mu.Lock()
	defer f.mu.Unlock()
	var body map[string]any
	if r.Body != nil && r.ContentLength != 0 {
		_ = json.NewDecoder(r.Body).Decode(&body)
	}
	p := strings.TrimPrefix(r.URL.Path, "/v2")
	f.calls = append(f.calls, call{r.Method, p, body})
	if r.Header.Get("Authorization") != "Bearer some-api-token" {
		w.WriteHeader(http.StatusUnauthorized)
		return
	}
	w.Header().Set("Content-Type", "application/json")

	if r.Method == http.MethodPost && p == "/deployments" {
		f.seq++
		id := fmt.Sprintf("00000000-0000-4000-8000-%012d", f.seq)
		typ, _ := body["type"].(string)
		d := map[string]any{
			"object": "deployment", "id": id, "type": typ, "name": "app", "status": "deploying",
			"raw": map[string]any{"service_id": body["service_id"], "environment_id": body["environment_id"], "project_id": body["project_id"]},
		}
		switch typ {
		case "web-service", "static-site":
			d["outputs"] = map[string]any{"url": "https://app.example.test", "internal_url": "app-private.example.test:8080"}
		case "private-service":
			// Also return a public address, which the private service
			// resource must never surface.
			d["outputs"] = map[string]any{"internal_url": "app-private.example.test:8080", "url": "https://leak.example.test"}
		}
		f.deps[id] = d
		w.WriteHeader(http.StatusCreated)
		_ = json.NewEncoder(w).Encode(d)
		return
	}
	parts := strings.Split(strings.TrimPrefix(p, "/deployments/"), "/")
	d, ok := f.deps[parts[0]]
	if !strings.HasPrefix(p, "/deployments/") || !ok {
		w.WriteHeader(http.StatusNotFound)
		return
	}
	action := ""
	if len(parts) > 1 {
		action = parts[1]
	}
	switch {
	case r.Method == http.MethodGet && action == "":
		d["status"] = "running"
		_ = json.NewEncoder(w).Encode(d)
	case r.Method == http.MethodGet && action == "status":
		_ = json.NewEncoder(w).Encode(map[string]any{"object": "deployment_status", "id": parts[0], "status": "running",
			"sphere_count": 1, "max_sphere_count": 1, "status_updated_at": "2026-10-12T10:00:00Z"})
	case r.Method == http.MethodPatch && action == "":
		if t, ok := body["type"].(string); ok && t != d["type"] {
			w.WriteHeader(http.StatusConflict)
			return
		}
		_ = json.NewEncoder(w).Encode(d)
	case r.Method == http.MethodPost && (action == "deploy" || action == "scale"):
		w.WriteHeader(http.StatusAccepted)
		_ = json.NewEncoder(w).Encode(d)
	case r.Method == http.MethodPut && action == "env-vars":
		_ = json.NewEncoder(w).Encode(map[string]any{"object": "env_vars", "env_vars": body["env_vars"], "secret_vars": map[string]any{}})
	case r.Method == http.MethodDelete && action == "":
		delete(f.deps, parts[0])
		w.WriteHeader(http.StatusNoContent)
	default:
		w.WriteHeader(http.StatusNotFound)
	}
}

// seed adds a deployment the API already has (as for an import) and
// returns its id.
func (f *fakeDeploymentsAPI) seed(typ string) string {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.seq++
	id := fmt.Sprintf("00000000-0000-4000-8000-%012d", f.seq)
	f.deps[id] = map[string]any{
		"object": "deployment", "id": id, "type": typ, "name": "app", "status": "running",
		"outputs": map[string]any{"url": "https://app.example.test", "internal_url": "app-private.example.test:8080"},
		"raw": map[string]any{"service_id": "44444444-4444-4444-4444-444444444444",
			"environment_id": "33333333-3333-3333-3333-333333333333", "project_id": "22222222-2222-2222-2222-222222222222"},
	}
	return id
}

// bodies returns the bodies of the calls matching method and a path suffix.
func (f *fakeDeploymentsAPI) bodies(method, suffix string) []map[string]any {
	f.mu.Lock()
	defer f.mu.Unlock()
	var out []map[string]any
	for _, c := range f.calls {
		if c.method == method && strings.HasSuffix(c.path, suffix) {
			out = append(out, c.body)
		}
	}
	return out
}

func (f *fakeDeploymentsAPI) count(method, suffix string) int { return len(f.bodies(method, suffix)) }

func (f *fakeDeploymentsAPI) reset() {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.calls = nil
}

// Closed bodies per type, from the API's spec. Fields the resources must
// never send are checked separately.
var (
	createKeys = map[string]string{
		"web-service":       "type name project_id environment_id env_id service_id plan_id plan port sphere_count spherelets image image_type image_provider image_username image_password image_url image_name image_pending build_source runtime_type language version build_command start_command dockerfile context git_url branch git_connection_id root_dir root_directory watch_paths build_args ci_config predeploy_command pipeline_tier health_check health_check_path health_check_port health_check_delay health_check_period health_check_timeout health_check_readiness_failures env_vars secret_vars environment_variables secret_variables auto_scaling_config autoscaling_enabled autoscaling_min autoscaling_max autoscaling_cpu_target network_access whitelist_ips custom_domain stor environment_spherestor",
		"private-service":   "type name project_id environment_id env_id service_id plan_id plan port sphere_count spherelets image image_type image_provider image_username image_password image_url image_name image_pending build_source runtime_type language version build_command start_command dockerfile context git_url branch git_connection_id root_dir root_directory watch_paths build_args ci_config predeploy_command pipeline_tier health_check health_check_path health_check_port health_check_delay health_check_period health_check_timeout health_check_readiness_failures env_vars secret_vars environment_variables secret_variables auto_scaling_config autoscaling_enabled autoscaling_min autoscaling_max autoscaling_cpu_target stor environment_spherestor",
		"background-worker": "type name project_id environment_id env_id service_id plan_id plan sphere_count spherelets image image_type image_provider image_username image_password image_url image_name command build_source runtime_type language version build_command start_command dockerfile context git_url branch git_connection_id root_dir root_directory watch_paths build_args ci_config env_vars secret_vars environment_variables secret_variables auto_scaling_config autoscaling_enabled autoscaling_min autoscaling_max autoscaling_cpu_target stor environment_spherestor",
		"cron-job":          "type name project_id environment_id env_id service_id plan_id plan schedule command image image_type image_provider image_username image_password image_url image_name build_source runtime_type language version build_command start_command dockerfile context git_url branch git_connection_id root_dir root_directory watch_paths build_args ci_config env_vars secret_vars environment_variables secret_variables",
		"static-site":       "type name project_id environment_id env_id service_id plan_id plan build_source git_url branch git_connection_id root_directory root_dir install_command build_command publish_directory spa env_vars secret_vars environment_variables secret_variables custom_domain",
	}
	changeKeys = map[string]string{
		"web-service":       "type port image health_check autoscaling networking build_config log_drains skip_redeploy deploy_policy trigger_on tag_pattern",
		"private-service":   "type port image health_check autoscaling build_config log_drains skip_redeploy deploy_policy trigger_on tag_pattern",
		"background-worker": "type skip_redeploy deploy_policy trigger_on tag_pattern",
		"cron-job":          "type schedule build_config log_drains skip_redeploy deploy_policy trigger_on tag_pattern",
		"static-site":       "type build_config networking skip_redeploy deploy_policy trigger_on tag_pattern",
	}
	// Server-only or refused fields; a resource never sends them. ci_config
	// is allowed only with its customer field git_repo_id (checked below).
	forbidden = []string{"tag", "is_build_complete", "webhook_id", "pipeline_tier", "plan_id", "plan",
		"custom_domain", "db_config"}
)

// checkClosed fails when body has a key outside allowed, or a forbidden key.
func checkClosed(t *testing.T, what string, body map[string]any, allowed string) {
	t.Helper()
	ok := map[string]bool{}
	for _, k := range strings.Fields(allowed) {
		ok[k] = true
	}
	var bad []string
	for k := range body {
		if !ok[k] {
			bad = append(bad, k)
		}
		if k == "ci_config" {
			ci, _ := body[k].(map[string]any)
			for ck := range ci {
				if ck != "git_repo_id" {
					bad = append(bad, "ci_config."+ck)
				}
			}
		}
		for _, f := range forbidden {
			if k == f {
				bad = append(bad, k+" (server-only)")
			}
		}
	}
	sort.Strings(bad)
	if len(bad) > 0 {
		t.Errorf("%s sends fields outside the closed body: %v (body %v)", what, bad, body)
	}
}
