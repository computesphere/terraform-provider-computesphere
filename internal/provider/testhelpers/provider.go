package testhelpers

import (
	"net/http"
	"os"
	"regexp"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/providerserver"
	"github.com/hashicorp/terraform-plugin-go/tfprotov6"
	"github.com/stretchr/testify/require"
	"gopkg.in/dnaeon/go-vcr.v3/cassette"
	"gopkg.in/dnaeon/go-vcr.v3/recorder"

	"github.com/computesphere/terraform-provider-computesphere/internal/provider"
)

func SetupRecordingProvider(t *testing.T, cassetteName string) map[string]func() (tfprotov6.ProviderServer, error) {
	return SetupRecordingProviderConfigureWait(t, cassetteName)
}

var tokenRegex = regexp.MustCompile(`(?i)Bearer [a-zA-Z0-9\-_\.]+`)

// testAccountID is the placeholder account id used in replay mode and scrubbed
// into recorded cassettes. It must be a valid UUID: the v2 SDK datasources parse
// account_id as a UUID before issuing a request, so a non-UUID placeholder (the
// former "some-account-id") fails validation during cassette replay.
const testAccountID = "11111111-1111-1111-1111-111111111111"

func scrubString(i *cassette.Interaction, from, to string) {
	i.Request.URL = strings.ReplaceAll(i.Request.URL, from, to)
	i.Request.Body = strings.ReplaceAll(i.Request.Body, from, to)
	i.Response.Body = strings.ReplaceAll(i.Response.Body, from, to)
}

func scrubRegex(i *cassette.Interaction, re *regexp.Regexp, to string) {
	i.Request.URL = re.ReplaceAllString(i.Request.URL, to)
	i.Request.Body = re.ReplaceAllString(i.Request.Body, to)
	i.Response.Body = re.ReplaceAllString(i.Response.Body, to)
}

func SetupRecordingProviderConfigureWait(t *testing.T, cassetteName string) map[string]func() (tfprotov6.ProviderServer, error) {
	mode := recorder.ModeRecordOnce
	if os.Getenv("UPDATE_RECORDINGS") == "true" {
		mode = recorder.ModeRecordOnly
	}

	r, err := recorder.NewWithOptions(&recorder.Options{
		CassetteName:       "testdata/" + cassetteName,
		Mode:               mode,
		SkipRequestLatency: true,
		RealTransport:      http.DefaultTransport,
	})
	require.NoError(t, err)
	t.Cleanup(func() {
		require.NoError(t, r.Stop())
	})

	replaceAuthHeader := func(i *cassette.Interaction) error {
		i.Request.Headers.Set("Authorization", "some-api-token")
		return nil
	}
	r.AddHook(replaceAuthHeader, recorder.AfterCaptureHook)

	replaceAccountID := func(i *cassette.Interaction) error {
		scrubString(i, os.Getenv("COMPUTESPHERE_ACCOUNT_ID"), testAccountID)
		return nil
	}
	r.AddHook(replaceAccountID, recorder.AfterCaptureHook)

	replaceToken := func(i *cassette.Interaction) error {
		scrubRegex(i, tokenRegex, "Bearer some-api-token")
		return nil
	}
	r.AddHook(replaceToken, recorder.AfterCaptureHook)

	removeHeaders := func(i *cassette.Interaction) error {
		i.Response.Headers.Del("Set-Cookie")
		return nil
	}
	r.AddHook(removeHeaders, recorder.AfterCaptureHook)

	var providerOpts []provider.ConfigFunc

	if r.IsRecording() {
		t.Log("Recording interactions for " + cassetteName)
		require.NotZero(t, os.Getenv("COMPUTESPHERE_API_URL"), "COMPUTESPHERE_API_URL must be set when recording")
		require.NotZero(t, os.Getenv("COMPUTESPHERE_API_TOKEN"), "COMPUTESPHERE_API_TOKEN must be set when recording")
		require.NotZero(t, os.Getenv("COMPUTESPHERE_ACCOUNT_ID"), "COMPUTESPHERE_ACCOUNT_ID must be set when recording")
		providerOpts = []provider.ConfigFunc{
			provider.WithHost(os.Getenv("COMPUTESPHERE_API_URL")),
			provider.WithAPIToken(os.Getenv("COMPUTESPHERE_API_TOKEN")),
			provider.WithAccountID(os.Getenv("COMPUTESPHERE_ACCOUNT_ID")),
		}
	} else {
		// Replay mode: fixed fake credentials; requests never reach the network.
		providerOpts = []provider.ConfigFunc{
			provider.WithHost("https://api.testing.computesphere.com"),
			provider.WithAPIToken("some-api-token"),
			provider.WithAccountID(testAccountID),
		}
	}

	// Inject the go-vcr client so both API clients record/replay through the
	// cassette instead of hitting the live API.
	if c := r.GetDefaultClient(); c != nil {
		providerOpts = append(providerOpts, provider.WithHTTPClient(c))
	}

	return map[string]func() (tfprotov6.ProviderServer, error){
		"computesphere": providerserver.NewProtocol6WithError(
			provider.New(
				"test",
				providerOpts...,
			)(),
		),
	}
}
