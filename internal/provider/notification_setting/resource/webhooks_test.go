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
	"github.com/hashicorp/terraform-plugin-testing/plancheck"
	"github.com/hashicorp/terraform-plugin-testing/terraform"

	"github.com/computesphere/terraform-provider-computesphere/internal/provider"
)

type fakeHook struct {
	URL         string `json:"url"`
	ContentType string `json:"content_type"`
	Token       string `json:"token"`
}

// fakeNotificationAPI keeps the saved settings like the API does: a PUT
// replaces the webhooks, a blank token keeps the stored token for that URL,
// and a read never returns a token.
type fakeNotificationAPI struct {
	mu      sync.Mutex
	hooks   []fakeHook
	billing bool
	puts    [][]fakeHook
}

func (f *fakeNotificationAPI) view() map[string]any {
	hooks := []map[string]any{}
	for _, h := range f.hooks {
		hooks = append(hooks, map[string]any{"url": h.URL, "content_type": h.ContentType, "token": "", "token_set": h.Token != ""})
	}
	return map[string]any{
		"object": "notification_settings", "id": "44444444-4444-4444-4444-444444444444",
		"user_id": "55555555-5555-5555-5555-555555555555", "billing": f.billing, "payment": false,
		"deployment": false, "activity": false, "invites": false, "inapp_enabled": false,
		"email_enabled": true, "webhook_enabled": true, "emails": []string{"user@example.com"},
		"webhooks": hooks,
	}
}

func (f *fakeNotificationAPI) handler() http.Handler {
	write := func(w http.ResponseWriter, v any) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(v)
	}
	mux := http.NewServeMux()
	mux.HandleFunc("GET /v2/notifications/settings", func(w http.ResponseWriter, r *http.Request) {
		f.mu.Lock()
		defer f.mu.Unlock()
		write(w, f.view())
	})
	mux.HandleFunc("PUT /v2/notifications/settings", func(w http.ResponseWriter, r *http.Request) {
		var body struct {
			Billing  bool       `json:"billing"`
			Webhooks []fakeHook `json:"webhooks"`
		}
		_ = json.NewDecoder(r.Body).Decode(&body)
		f.mu.Lock()
		defer f.mu.Unlock()
		f.puts = append(f.puts, body.Webhooks)
		f.billing = body.Billing
		stored := map[string]string{}
		for _, h := range f.hooks {
			stored[h.URL] = h.Token
		}
		next := []fakeHook{}
		for _, h := range body.Webhooks {
			if h.Token == "" {
				h.Token = stored[h.URL]
			}
			next = append(next, h)
		}
		f.hooks = next
		write(w, f.view())
	})
	return mux
}

// lastPut checks the webhooks the last PUT sent.
func (f *fakeNotificationAPI) lastPut(want ...fakeHook) resource.TestCheckFunc {
	return func(*terraform.State) error {
		f.mu.Lock()
		defer f.mu.Unlock()
		if len(f.puts) == 0 {
			return fmt.Errorf("no PUT sent")
		}
		got := f.puts[len(f.puts)-1]
		if fmt.Sprint(got) != fmt.Sprint(want) {
			return fmt.Errorf("PUT webhooks = %v, want %v", got, want)
		}
		return nil
	}
}

func (f *fakeNotificationAPI) stored(want ...fakeHook) resource.TestCheckFunc {
	return func(*terraform.State) error {
		f.mu.Lock()
		defer f.mu.Unlock()
		if fmt.Sprint(f.hooks) != fmt.Sprint(want) {
			return fmt.Errorf("stored webhooks = %v, want %v", f.hooks, want)
		}
		return nil
	}
}

func setupFakeNotificationAPI(t *testing.T) (map[string]func() (tfprotov6.ProviderServer, error), *fakeNotificationAPI) {
	t.Helper()
	f := &fakeNotificationAPI{hooks: []fakeHook{{URL: "https://hooks.example.com/console", ContentType: "json", Token: "console-token"}}}
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

// Without webhooks in the configuration, the webhooks saved in the console are
// sent back unchanged (their tokens kept); with webhooks set, they are managed:
// sent as configured, a left-out token keeps the stored one, and an empty list
// removes them.
func TestAccNotificationSettingResource_webhooks(t *testing.T) {
	factories, f := setupFakeNotificationAPI(t)
	const addr = "computesphere_notification_setting.example"
	cfg := func(extra string) string {
		return `resource "computesphere_notification_setting" "example" {
  email_enabled   = true
  webhook_enabled = true
  emails          = ["user@example.com"]
` + extra + `
}`
	}
	console := fakeHook{URL: "https://hooks.example.com/console", ContentType: "json", Token: "console-token"}
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: factories,
		Steps: []resource.TestStep{
			{
				Config: cfg(""),
				Check: resource.ComposeAggregateTestCheckFunc(
					f.lastPut(fakeHook{URL: console.URL, ContentType: "json"}),
					f.stored(console),
					resource.TestCheckNoResourceAttr(addr, "webhooks"),
				),
			},
			{
				// Another change leaves the console's webhook alone.
				Config: cfg(`  billing = true`),
				Check: resource.ComposeAggregateTestCheckFunc(
					f.lastPut(fakeHook{URL: console.URL, ContentType: "json"}),
					f.stored(console),
				),
			},
			{
				Config: cfg(`  billing = true
  webhooks = [
    { url = "https://hooks.example.com/console" },
    { url = "https://hooks.example.com/new", content_type = "text", token = "new-token" },
  ]`),
				Check: resource.ComposeAggregateTestCheckFunc(
					f.lastPut(fakeHook{URL: console.URL, ContentType: "json"},
						fakeHook{URL: "https://hooks.example.com/new", ContentType: "text", Token: "new-token"}),
					f.stored(console, fakeHook{URL: "https://hooks.example.com/new", ContentType: "text", Token: "new-token"}),
					resource.TestCheckResourceAttr(addr, "webhooks.1.token", "new-token"),
				),
			},
			{
				// The API never returns tokens; the configured one stays in state.
				Config: cfg(`  billing = true
  webhooks = [
    { url = "https://hooks.example.com/console" },
    { url = "https://hooks.example.com/new", content_type = "text", token = "new-token" },
  ]`),
				ConfigPlanChecks: resource.ConfigPlanChecks{PreApply: []plancheck.PlanCheck{plancheck.ExpectEmptyPlan()}},
			},
			{
				Config: cfg(`  billing = true
  webhooks = []`),
				Check: resource.ComposeAggregateTestCheckFunc(
					f.lastPut(),
					f.stored(),
					resource.TestCheckResourceAttr(addr, "webhooks.#", "0"),
				),
			},
		},
	})
}

// A scoped API token gets 403 on notification settings; the error says to use
// a full-access token.
func TestAccNotificationSettingResource_scopedToken(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/problem+json")
		w.WriteHeader(http.StatusForbidden)
		_, _ = w.Write([]byte(`{"status":403,"code":"forbidden","title":"Forbidden"}`))
	}))
	t.Cleanup(srv.Close)
	factories := map[string]func() (tfprotov6.ProviderServer, error){
		"computesphere": providerserver.NewProtocol6WithError(provider.New("test",
			provider.WithHost(srv.URL+"/v2"),
			provider.WithAPIToken("some-api-token"),
			provider.WithAccountID("11111111-1111-1111-1111-111111111111"),
			provider.WithHTTPClient(srv.Client()),
		)()),
	}
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: factories,
		Steps: []resource.TestStep{{
			Config:      `resource "computesphere_notification_setting" "example" { email_enabled = true }`,
			ExpectError: regexp.MustCompile(`full-access`),
		}},
	})
}
