package testhelpers

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/providerserver"
	"github.com/hashicorp/terraform-plugin-go/tfprotov6"

	"github.com/computesphere/terraform-provider-computesphere/internal/provider"
)

// FakeAccountID is the account id the fake-API provider is configured with.
const FakeAccountID = testAccountID

// SetupFakeAPIProvider serves handler as the v2 API on a local test server
// and returns provider factories pointed at it. Use it for unit tests of
// resources whose API routes can't be recorded yet: the handler stands in
// for the API, so the test runs offline under resource.UnitTest.
func SetupFakeAPIProvider(t *testing.T, handler http.Handler) map[string]func() (tfprotov6.ProviderServer, error) {
	t.Helper()
	srv := httptest.NewServer(handler)
	t.Cleanup(srv.Close)
	return map[string]func() (tfprotov6.ProviderServer, error){
		"computesphere": providerserver.NewProtocol6WithError(
			provider.New(
				"test",
				provider.WithHost(srv.URL+"/v2"),
				provider.WithAPIToken("some-api-token"),
				provider.WithAccountID(FakeAccountID),
				provider.WithHTTPClient(srv.Client()),
			)(),
		),
	}
}
