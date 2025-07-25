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
		scrubString(i, os.Getenv("COMPUTESPHERE_ACCOUNT_ID"), "some-account-id")
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

	providerOpts := []provider.ConfigFunc{
		provider.WithHost(os.Getenv("COMPUTESPHERE_API_URL")),
		provider.WithAPIToken(os.Getenv("COMPUTESPHERE_API_TOKEN")),
		provider.WithAccountID(os.Getenv("COMPUTESPHERE_ACCOUNT_ID")),
	}

	if r.GetDefaultClient() != nil {
		// If go-vcr is recording or replaying, use its client
		// (Assumes provider supports HTTP client injection; add provider.WithHTTPClient if available)
	}

	if r.IsRecording() {
		t.Log("Recording interactions for " + cassetteName)
		require.NotZero(t, os.Getenv("COMPUTESPHERE_API_URL"), "COMPUTESPHERE_API_URL must be set when recording")
		require.NotZero(t, os.Getenv("COMPUTESPHERE_API_TOKEN"), "COMPUTESPHERE_API_TOKEN must be set when recording")
		require.NotZero(t, os.Getenv("COMPUTESPHERE_ACCOUNT_ID"), "COMPUTESPHERE_ACCOUNT_ID must be set when recording")
	} else {
		// Optionally set fake values for replay mode
		providerOpts = []provider.ConfigFunc{
			provider.WithHost("https://api.testing.computesphere.com/v1"),
			provider.WithAPIToken("some-api-token"),
			provider.WithAccountID("some-account-id"),
		}
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
