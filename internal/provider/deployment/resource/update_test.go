package provider_test

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
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
	depAddr   = "computesphere_deployment.example"
	envID     = "b3669d6a-d89e-4037-873a-65903f3f3ced"
	otherEnv  = "c4770e7b-e9af-4148-984b-76a14f4d4d0e"
	serviceID = "aaaaaaaa-0000-4000-8000-000000000001"
	otherSvc  = "aaaaaaaa-0000-4000-8000-000000000002"
)

// fakeDeployAPI is an in-process stand-in for the deployment endpoints. It
// logs every request that changes something.
type fakeDeployAPI struct {
	mu   sync.Mutex
	seq  int
	deps map[string]bool
	sent []string // "METHOD /path body", writes only
}

func (f *fakeDeployAPI) reset() {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.sent = nil
}

// requests returns the logged writes that start with prefix.
func (f *fakeDeployAPI) requests(prefix string) []string {
	f.mu.Lock()
	defer f.mu.Unlock()
	var out []string
	for _, r := range f.sent {
		if strings.HasPrefix(r, prefix) {
			out = append(out, r)
		}
	}
	return out
}

func (f *fakeDeployAPI) handler() http.Handler {
	view := func(id string) map[string]any {
		return map[string]any{"object": "deployment", "id": id, "name": "example", "type": "web-service", "status": "Running"}
	}
	write := func(w http.ResponseWriter, code int, v any) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(code)
		_ = json.NewEncoder(w).Encode(v)
	}
	mux := http.NewServeMux()
	mux.HandleFunc("POST /v2/deployments", func(w http.ResponseWriter, r *http.Request) {
		f.mu.Lock()
		f.seq++
		id := fmt.Sprintf("bbbbbbbb-0000-4000-8000-%012d", f.seq)
		f.deps[id] = true
		f.mu.Unlock()
		write(w, http.StatusCreated, view(id))
	})
	mux.HandleFunc("GET /v2/deployments/{id}", func(w http.ResponseWriter, r *http.Request) {
		f.mu.Lock()
		ok := f.deps[r.PathValue("id")]
		f.mu.Unlock()
		if !ok {
			write(w, http.StatusNotFound, map[string]any{"status": 404, "title": "Not Found"})
			return
		}
		write(w, http.StatusOK, view(r.PathValue("id")))
	})
	mux.HandleFunc("PATCH /v2/deployments/{id}", func(w http.ResponseWriter, r *http.Request) {
		write(w, http.StatusOK, view(r.PathValue("id")))
	})
	mux.HandleFunc("PUT /v2/deployments/{id}/env-vars", func(w http.ResponseWriter, r *http.Request) {
		write(w, http.StatusOK, view(r.PathValue("id")))
	})
	mux.HandleFunc("POST /v2/deployments/{id}/deploy", func(w http.ResponseWriter, r *http.Request) {
		write(w, http.StatusAccepted, map[string]any{"object": "deployment", "id": r.PathValue("id"), "status": "Redeploying"})
	})
	mux.HandleFunc("GET /v2/deployments/{id}/status", func(w http.ResponseWriter, r *http.Request) {
		write(w, http.StatusOK, map[string]any{"object": "deployment_status", "id": r.PathValue("id"), "status": "Running"})
	})
	mux.HandleFunc("DELETE /v2/deployments/{id}", func(w http.ResponseWriter, r *http.Request) {
		f.mu.Lock()
		delete(f.deps, r.PathValue("id"))
		f.mu.Unlock()
		w.WriteHeader(http.StatusNoContent)
	})
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			var raw json.RawMessage
			_ = json.NewDecoder(r.Body).Decode(&raw)
			f.mu.Lock()
			f.sent = append(f.sent, strings.TrimSpace(r.Method+" "+r.URL.Path+" "+string(raw)))
			f.mu.Unlock()
		}
		mux.ServeHTTP(w, r)
	})
}

func setupFakeDeployAPI(t *testing.T) (map[string]func() (tfprotov6.ProviderServer, error), *fakeDeployAPI) {
	t.Helper()
	f := &fakeDeployAPI{deps: map[string]bool{}}
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

// depConfig is a deployment with the given extra attribute lines.
func depConfig(service, env string, attrs ...string) string {
	return fmt.Sprintf(`
resource "computesphere_deployment" "example" {
  service_id     = %q
  project_id     = "13316e7e-21d7-43c8-9ee7-69674510ceeb"
  environment_id = %q
  type           = "web-service"
  %s
}
`, service, env, strings.Join(attrs, "\n  "))
}

const (
	aImage    = `image = "nginx:1.27"`
	aImage2   = `image = "nginx:1.28"`
	aPort     = `port = 80`
	aCount    = `sphere_count = 1`
	aEnv      = `env_vars = { A = "1" }`
	aSecret   = `secret_vars = { S = "s3cret" }`
	aArgs     = `build_args = { B = "1" }`
	aNoWait   = `wait_for_ready = false`
	aWait     = `wait_for_ready = true`
	aPlanID   = `plan_id = "00000000-0000-4000-8000-0000000000a1"`
	aPlanID2  = `plan_id = "00000000-0000-4000-8000-0000000000a2"`
	aRegistry = `image_type = "private"
  image_provider = "docker"
  image_username = "user"
  image_password = "pass"
  image_url = "registry.example.com"`
)

// expectSent checks how many writes starting with prefix the step sent.
func expectSent(f *fakeDeployAPI, prefix string, want int) resource.TestCheckFunc {
	return func(*terraform.State) error {
		if got := f.requests(prefix); len(got) != want {
			return fmt.Errorf("%d requests starting %q, want %d: %v", len(got), prefix, want, f.requests(""))
		}
		return nil
	}
}

// expectBody checks the JSON body of the only write starting with prefix.
func expectBody(f *fakeDeployAPI, prefix string, check func(map[string]any) error) resource.TestCheckFunc {
	return func(*terraform.State) error {
		got := f.requests(prefix)
		if len(got) != 1 {
			return fmt.Errorf("%d requests starting %q, want 1: %v", len(got), prefix, f.requests(""))
		}
		var body map[string]any
		if err := json.Unmarshal([]byte(strings.TrimPrefix(got[0], strings.SplitN(got[0], " {", 2)[0]+" ")), &body); err != nil {
			return fmt.Errorf("body of %q: %w", got[0], err)
		}
		return check(body)
	}
}

// inPlace is the check set for an update: one deployment, no delete or create.
func inPlace(f *fakeDeployAPI) resource.TestCheckFunc {
	return resource.ComposeAggregateTestCheckFunc(
		expectSent(f, "DELETE ", 0),
		expectSent(f, "POST /v2/deployments {", 0),
		expectSent(f, "POST /v2/deployments/", 1), // the deploy
	)
}

func replaced(f *fakeDeployAPI) resource.TestCheckFunc {
	return resource.ComposeAggregateTestCheckFunc(
		expectSent(f, "DELETE /v2/deployments/", 1),
		expectSent(f, "POST /v2/deployments {", 1),
	)
}

func planAction(a plancheck.ResourceActionType) resource.ConfigPlanChecks {
	return resource.ConfigPlanChecks{PreApply: []plancheck.PlanCheck{plancheck.ExpectResourceAction(depAddr, a)}}
}

// Changes the update API can apply go through it, on the same deployment;
// removing a variable map clears it; a wait_for_ready change sends nothing.
func TestAccDeploymentResource_updatesInPlace(t *testing.T) {
	factories, f := setupFakeDeployAPI(t)
	base := []string{aPort, aCount, aNoWait, aRegistry}
	cfg := func(attrs ...string) string {
		return depConfig(serviceID, envID, append(append([]string{}, base...), attrs...)...)
	}
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: factories,
		Steps: []resource.TestStep{
			{
				PreConfig: f.reset,
				Config:    cfg(aImage, aEnv, aSecret, aArgs),
				Check:     resource.TestCheckResourceAttrSet(depAddr, "id"),
			},
			{
				PreConfig:        f.reset,
				Config:           cfg(aImage2, aEnv, aSecret, aArgs),
				ConfigPlanChecks: planAction(plancheck.ResourceActionUpdate),
				Check: resource.ComposeAggregateTestCheckFunc(
					inPlace(f),
					expectBody(f, "PATCH /v2/deployments/", func(b map[string]any) error {
						img, _ := b["image"].(map[string]any)
						if img["name"] != "nginx:1.28" || img["password"] != "pass" {
							return fmt.Errorf("image = %v, want the new name with the registry auth", img)
						}
						return nil
					}),
					expectSent(f, "PUT ", 0),
				),
			},
			{
				Config:           cfg(aImage2, aEnv, aSecret, aArgs),
				ConfigPlanChecks: resource.ConfigPlanChecks{PreApply: []plancheck.PlanCheck{plancheck.ExpectEmptyPlan()}},
			},
			{
				// Removing env_vars clears them and leaves the secrets alone.
				PreConfig:        f.reset,
				Config:           cfg(aImage2, aSecret, aArgs),
				ConfigPlanChecks: planAction(plancheck.ResourceActionUpdate),
				Check: resource.ComposeAggregateTestCheckFunc(
					inPlace(f),
					expectSent(f, "PATCH ", 0),
					expectBody(f, "PUT /v2/deployments/", func(b map[string]any) error {
						if v, ok := b["env_vars"].(map[string]any); !ok || len(v) != 0 {
							return fmt.Errorf("env_vars = %v, want {}", b["env_vars"])
						}
						if _, ok := b["secret_vars"]; ok {
							return fmt.Errorf("secret_vars sent: %v", b["secret_vars"])
						}
						return nil
					}),
				),
			},
			{
				// Removing secret_vars clears them and leaves env_vars alone.
				PreConfig:        f.reset,
				Config:           cfg(aImage2, aArgs, `env_vars = { C = "3" }`),
				ConfigPlanChecks: planAction(plancheck.ResourceActionUpdate),
				Check: resource.ComposeAggregateTestCheckFunc(
					inPlace(f),
					expectBody(f, "PUT /v2/deployments/", func(b map[string]any) error {
						if v, ok := b["secret_vars"].(map[string]any); !ok || len(v) != 0 {
							return fmt.Errorf("secret_vars = %v, want {}", b["secret_vars"])
						}
						if v, _ := b["env_vars"].(map[string]any); v["C"] != "3" {
							return fmt.Errorf("env_vars = %v, want {C: 3}", b["env_vars"])
						}
						return nil
					}),
				),
			},
			{
				// Removing build_args sends an explicit null, which clears them.
				PreConfig:        f.reset,
				Config:           cfg(aImage2, `env_vars = { C = "3" }`),
				ConfigPlanChecks: planAction(plancheck.ResourceActionUpdate),
				Check: resource.ComposeAggregateTestCheckFunc(
					inPlace(f),
					expectSent(f, "PUT ", 0),
					expectBody(f, "PATCH /v2/deployments/", func(b map[string]any) error {
						bc, _ := b["build_config"].(map[string]any)
						if v, ok := bc["build_args"]; !ok || v != nil {
							return fmt.Errorf("build_config = %v, want build_args: null", b["build_config"])
						}
						return nil
					}),
				),
			},
			{
				// wait_for_ready is a provider setting: changing it sends nothing.
				PreConfig: f.reset,
				Config: depConfig(serviceID, envID, aPort, aCount, aWait, aRegistry, aImage2,
					`env_vars = { C = "3" }`),
				ConfigPlanChecks: planAction(plancheck.ResourceActionUpdate),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(depAddr, "wait_for_ready", "true"),
					expectSent(f, "", 0),
				),
			},
		},
	})
}

// Changes the update API can't apply still replace the deployment.
func TestAccDeploymentResource_replaces(t *testing.T) {
	factories, f := setupFakeDeployAPI(t)
	step := func(config string) resource.TestStep {
		return resource.TestStep{
			PreConfig:        f.reset,
			Config:           config,
			ConfigPlanChecks: planAction(plancheck.ResourceActionDestroyBeforeCreate),
			Check:            replaced(f),
		}
	}
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: factories,
		Steps: []resource.TestStep{
			{Config: depConfig(serviceID, envID, aImage, aPort, aCount, aNoWait, aPlanID, aRegistry)},
			step(depConfig(serviceID, otherEnv, aImage, aPort, aCount, aNoWait, aPlanID, aRegistry)), // environment_id
			step(depConfig(otherSvc, otherEnv, aImage, aPort, aCount, aNoWait, aPlanID, aRegistry)),  // service_id
			step(depConfig(otherSvc, otherEnv, aImage, aPort, aCount, aNoWait, aPlanID2, aRegistry)), // plan_id
			step(depConfig(otherSvc, otherEnv, aPort, aCount, aNoWait, aPlanID2, aRegistry)),         // image removed
			step(depConfig(otherSvc, otherEnv, aCount, aNoWait, aPlanID2, aRegistry)),                // port removed
			step(depConfig(otherSvc, otherEnv, aNoWait, aPlanID2, aRegistry)),                        // sphere_count removed
			step(depConfig(otherSvc, otherEnv, aNoWait, aPlanID2)),                                   // registry auth removed
		},
	})
}
