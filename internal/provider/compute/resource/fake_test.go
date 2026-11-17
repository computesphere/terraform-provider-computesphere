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
	// rejectImage answers 422 to a change body with an image object, as the
	// API does until a type's backend takes image changes.
	rejectImage bool
	t           *testing.T
	mu          sync.Mutex
	deps        map[string]map[string]any
	// builds is the build state of a git-built deployment, as the API keeps
	// it: "running" while a build the create started runs, "requested" while
	// one started by POST /builds runs (it finishes when its record reads
	// succeeded); a /deploy is refused until the first build has finished.
	// "none" when no build was ever started, "lost"
	// when the create's git source was dropped. A read finishes a running
	// build. Image deployments have no entry.
	builds map[string]string
	// failBuild makes every build fail. buildReads counts reads of each
	// build record: the first answers dispatched, the next its result,
	// while the deployment's own status keeps reading running.
	failBuild  bool
	buildEnd   string // when set, the status a build ends in instead
	buildReads map[string]int
	seq        int
	calls      []call
	// nameHold is how many creates a deleted service's name answers 409
	// to in its environment, as the API holds it until the delete
	// finishes. taken answers 409 to every create: a name in use.
	nameHold int
	held     map[string]int
	taken    bool
}

type call struct {
	method, path string
	body         map[string]any
}

func newFakeDeploymentsAPI(t *testing.T) *fakeDeploymentsAPI {
	return &fakeDeploymentsAPI{t: t, deps: map[string]map[string]any{}, builds: map[string]string{}, buildReads: map[string]int{}, held: map[string]int{}}
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
		key := fmt.Sprint(body["environment_id"], "/", body["name"])
		if f.taken || f.held[key] > 0 {
			f.held[key]--
			conflict(w, fmt.Sprintf("A resource named %q in this environment is still being deleted.", body["name"]))
			return
		}
		f.seq++
		id := fmt.Sprintf("00000000-0000-4000-8000-%012d", f.seq)
		typ, _ := body["type"].(string)
		name, serviceID := "app", body["service_id"]
		if serviceID == nil {
			// A standalone create: the API creates the service with the name.
			name, _ = body["name"].(string)
			serviceID = fmt.Sprintf("55555555-5555-4555-8555-%012d", f.seq)
		}
		d := map[string]any{
			"object": "deployment", "id": id, "type": typ, "name": name, "status": "deploying",
			"raw": map[string]any{"service_id": serviceID, "env_id": body["environment_id"], "project_id": body["project_id"],
				// The read always carries ci_config.git_repo_id, "" when unset.
				"ci_config": map[string]any{"git_repo_id": repoID(body)}},
		}
		switch typ {
		case "web-service", "static-site":
			d["outputs"] = map[string]any{"url": "https://app.example.test", "internal_url": "app-private.example.test:8080"}
		case "private-service":
			// Also return a public address, which the private service
			// resource must never surface.
			d["outputs"] = map[string]any{"internal_url": "app-private.example.test:8080", "url": "https://leak.example.test"}
		}
		if it, ok := body["instance_type"].(string); ok {
			if raw, ok := d["raw"].(map[string]any); ok {
				raw["instance_type"] = resolveSize(it)
			}
		}
		f.deps[id] = d
		if st := createBuildState(typ, body); st != "" {
			f.builds[id] = st
		}
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
	if r.Method == http.MethodGet && f.builds[parts[0]] == "running" {
		f.builds[parts[0]] = "done"
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
		if _, ok := body["image"]; ok && f.rejectImage {
			w.Header().Set("Content-Type", "application/problem+json")
			w.WriteHeader(http.StatusUnprocessableEntity)
			_, _ = w.Write([]byte(`{"title":"Unprocessable","status":422,"detail":"image changes aren't available for this type yet"}`))
			return
		}
		if bc, ok := body["build_config"].(map[string]any); ok && bc["git_url"] != nil && f.builds[parts[0]] == "" {
			// An image service switched to git: its manifest is saved, and
			// the image keeps serving until a build replaces it.
			f.builds[parts[0]] = "done"
		}
		_ = json.NewEncoder(w).Encode(d)
	case r.Method == http.MethodPost && action == "scale" && d["type"] == "cron-job":
		// A cron job isn't scalable.
		problem(w, http.StatusUnprocessableEntity, "cron-job does not support scale")
	case r.Method == http.MethodPost && action == "builds":
		if f.builds[parts[0]] == "lost" || f.builds[parts[0]] == "" {
			problem(w, http.StatusBadRequest, "no saved manifest to build from")
			return
		}
		f.builds[parts[0]] = "requested"
		f.seq++
		w.WriteHeader(http.StatusAccepted)
		_ = json.NewEncoder(w).Encode(map[string]any{"object": "build", "status": "dispatched",
			"id": fmt.Sprintf("bbbbbbbb-0000-4000-8000-%012d", f.seq)})
	case r.Method == http.MethodGet && action == "builds" && len(parts) == 3:
		f.buildReads[parts[2]]++
		st := "dispatched"
		switch {
		case f.buildReads[parts[2]] < 2:
		case f.buildEnd != "":
			st = f.buildEnd
		case f.failBuild:
			st = "failed"
		default:
			st = "succeeded"
			f.builds[parts[0]] = "done"
		}
		_ = json.NewEncoder(w).Encode(map[string]any{"object": "build", "id": parts[2], "status": st})
	case r.Method == http.MethodPost && action == "scale":
		if it, ok := body["instance_type"].(string); ok {
			if raw, ok := d["raw"].(map[string]any); ok {
				raw["instance_type"] = resolveSize(it)
			}
		}
		w.WriteHeader(http.StatusAccepted)
		_ = json.NewEncoder(w).Encode(d)
	case r.Method == http.MethodPost && action == "deploy" && f.builds[parts[0]] != "" && f.builds[parts[0]] != "done":
		// Nothing built yet: the API refuses a redeploy before the first build.
		problem(w, http.StatusBadRequest, "no built image available — trigger a new build to deploy this service")
	case r.Method == http.MethodPost && action == "deploy":
		w.WriteHeader(http.StatusAccepted)
		_ = json.NewEncoder(w).Encode(d)
	case r.Method == http.MethodPut && action == "env-vars":
		_ = json.NewEncoder(w).Encode(map[string]any{"object": "env_vars", "env_vars": body["env_vars"], "secret_vars": map[string]any{}})
	case r.Method == http.MethodDelete && action == "":
		if raw, ok := d["raw"].(map[string]any); ok && f.nameHold > 0 {
			f.held[fmt.Sprint(raw["env_id"], "/", d["name"])] = f.nameHold
		}
		delete(f.deps, parts[0])
		w.WriteHeader(http.StatusNoContent)
	default:
		w.WriteHeader(http.StatusNotFound)
	}
}

func conflict(w http.ResponseWriter, detail string) {
	w.Header().Set("Content-Type", "application/problem+json")
	w.WriteHeader(http.StatusConflict)
	_ = json.NewEncoder(w).Encode(map[string]any{"title": "Conflict", "status": 409, "code": "conflict", "detail": detail})
}

func problem(w http.ResponseWriter, status int, detail string) {
	w.Header().Set("Content-Type", "application/problem+json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(map[string]any{"title": http.StatusText(status), "status": status, "detail": detail})
}

// createBuildState is the build state a create leaves, as the API decides it:
// a static site and a flat git create start their first build; a create
// with ci_config is read from ci_config alone and starts none, so one whose
// ci_config has no git_url loses its git source.
func createBuildState(typ string, body map[string]any) string {
	ci, nested := body["ci_config"].(map[string]any)
	flat := body["git_url"] != nil || body["build_source"] == "git"
	switch {
	case typ == "static-site":
		return "running"
	case nested && ci["git_url"] != nil:
		return "none"
	case nested && flat:
		return "lost"
	case flat:
		return "running"
	}
	return ""
}

// resolveSize stores a size alias as its code, as the API does.
func resolveSize(v string) string {
	if v == "FLX" {
		return "sl-0.5c-512m"
	}
	return v
}

// seed adds a deployment the API already has (as for an import) and
// returns its id.
func repoID(body map[string]any) string {
	ci, _ := body["ci_config"].(map[string]any)
	v, _ := ci["git_repo_id"].(string)
	return v
}

func (f *fakeDeploymentsAPI) seed(typ string) string { return f.seedWithRepo(typ, "") }

// seedWithRepo is seed for a deployment whose git_repo_id is repo.
func (f *fakeDeploymentsAPI) seedWithRepo(typ, repo string) string {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.seq++
	id := fmt.Sprintf("00000000-0000-4000-8000-%012d", f.seq)
	f.deps[id] = map[string]any{
		"object": "deployment", "id": id, "type": typ, "name": "app", "status": "running",
		"outputs": map[string]any{"url": "https://app.example.test", "internal_url": "app-private.example.test:8080"},
		"raw": map[string]any{"service_id": "44444444-4444-4444-4444-444444444444",
			"env_id": "33333333-3333-3333-3333-333333333333", "project_id": "22222222-2222-2222-2222-222222222222",
			"ci_config": map[string]any{"git_repo_id": repo}},
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
		"web-service":       "type name project_id environment_id env_id service_id plan_id plan instance_type port sphere_count spherelets image image_type image_provider image_username image_password image_url image_name image_pending build_source runtime_type language version build_command start_command dockerfile context git_url branch git_connection_id root_dir root_directory watch_paths build_args ci_config predeploy_command pipeline_tier health_check health_check_path health_check_port health_check_delay health_check_period health_check_timeout health_check_readiness_failures env_vars secret_vars environment_variables secret_variables auto_scaling_config autoscaling_enabled autoscaling_min autoscaling_max autoscaling_cpu_target network_access whitelist_ips custom_domain stor environment_spherestor",
		"private-service":   "type name project_id environment_id env_id service_id plan_id plan instance_type port sphere_count spherelets image image_type image_provider image_username image_password image_url image_name image_pending build_source runtime_type language version build_command start_command dockerfile context git_url branch git_connection_id root_dir root_directory watch_paths build_args ci_config predeploy_command pipeline_tier health_check health_check_path health_check_port health_check_delay health_check_period health_check_timeout health_check_readiness_failures env_vars secret_vars environment_variables secret_variables auto_scaling_config autoscaling_enabled autoscaling_min autoscaling_max autoscaling_cpu_target stor environment_spherestor",
		"background-worker": "type name project_id environment_id env_id service_id plan_id plan instance_type sphere_count spherelets image image_type image_provider image_username image_password image_url image_name command build_source runtime_type language version build_command start_command dockerfile context git_url branch git_connection_id root_dir root_directory watch_paths build_args ci_config env_vars secret_vars environment_variables secret_variables auto_scaling_config autoscaling_enabled autoscaling_min autoscaling_max autoscaling_cpu_target stor environment_spherestor",
		"cron-job":          "type name project_id environment_id env_id service_id plan_id plan instance_type schedule command image image_type image_provider image_username image_password image_url image_name build_source runtime_type language version build_command start_command dockerfile context git_url branch git_connection_id root_dir root_directory watch_paths build_args ci_config env_vars secret_vars environment_variables secret_variables",
		"static-site":       "type name project_id environment_id env_id service_id plan_id plan build_source git_url branch git_connection_id root_directory root_dir install_command build_command publish_directory spa network_access whitelist_ips ci_config env_vars secret_vars environment_variables secret_variables custom_domain",
	}
	changeKeys = map[string]string{
		"web-service":       "type port image health_check autoscaling networking build_config log_drains skip_redeploy deploy_policy trigger_on tag_pattern",
		"private-service":   "type port image health_check autoscaling build_config log_drains skip_redeploy deploy_policy trigger_on tag_pattern",
		"background-worker": "type image autoscaling build_config log_drains skip_redeploy deploy_policy trigger_on tag_pattern",
		"cron-job":          "type schedule image build_config log_drains skip_redeploy deploy_policy trigger_on tag_pattern",
		"static-site":       "type build_config networking skip_redeploy deploy_policy trigger_on tag_pattern",
	}
	// Server-only or refused fields; a resource never sends them. ci_config
	// is allowed only with its customer field git_repo_id (checked below).
	forbidden = []string{"tag", "is_build_complete", "webhook_id", "pipeline_tier", "plan_id", "plan",
		"custom_domain", "db_config"}
)

// checkClosed fails when body has a key outside allowed, or a forbidden key.
// ciConfigKeys are the customer keys of the nested git source (the spec's
// ComputeCIConfig); tag, is_build_complete and webhook_id are server-only.
var ciConfigKeys = map[string]bool{}

func init() {
	for _, k := range strings.Fields("git_url git_branch git_connection_id git_repo_id git_username git_password " +
		"type working_dir base_image base_version build_script start_script install_script publish_directory " +
		"build_type dockerfile docker_context root_dir watch_paths build_args webhook_enabled spa") {
		ciConfigKeys[k] = true
	}
}

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
				if !ciConfigKeys[ck] {
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
