package provider

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"strings"

	csv2 "github.com/computesphere/computesphere-go"
	cstypes "github.com/computesphere/terraform-provider-computesphere/internal/provider/types"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/function"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/provider"
	"github.com/hashicorp/terraform-plugin-framework/provider/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"

	// Resource imports

	alertresource "github.com/computesphere/terraform-provider-computesphere/internal/provider/alert/resource"
	apitokenresource "github.com/computesphere/terraform-provider-computesphere/internal/provider/api_token/resource"
	customdomainresource "github.com/computesphere/terraform-provider-computesphere/internal/provider/custom_domain/resource"
	deploymentresource "github.com/computesphere/terraform-provider-computesphere/internal/provider/deployment/resource"
	environmentresource "github.com/computesphere/terraform-provider-computesphere/internal/provider/environment/resource"
	guardrailresource "github.com/computesphere/terraform-provider-computesphere/internal/provider/guardrail/resource"
	notificationresource "github.com/computesphere/terraform-provider-computesphere/internal/provider/notification_setting/resource"
	projectresource "github.com/computesphere/terraform-provider-computesphere/internal/provider/project/resource"
	serviceresource "github.com/computesphere/terraform-provider-computesphere/internal/provider/service/resource"
	teamresource "github.com/computesphere/terraform-provider-computesphere/internal/provider/team/resource"

	// Datasource imports

	alertdatasource "github.com/computesphere/terraform-provider-computesphere/internal/provider/alert/datasource"
	environmentdatasource "github.com/computesphere/terraform-provider-computesphere/internal/provider/environment/datasource"
	guardraildatasource "github.com/computesphere/terraform-provider-computesphere/internal/provider/guardrail/datasource"
	notificationdatasource "github.com/computesphere/terraform-provider-computesphere/internal/provider/notification_setting/datasource"
	plandatasource "github.com/computesphere/terraform-provider-computesphere/internal/provider/plan/datasource"
	projectdatasource "github.com/computesphere/terraform-provider-computesphere/internal/provider/project/datasource"
	regiondatasource "github.com/computesphere/terraform-provider-computesphere/internal/provider/region/datasource"
	servicedatasource "github.com/computesphere/terraform-provider-computesphere/internal/provider/service/datasource"
	subscriptiondatasource "github.com/computesphere/terraform-provider-computesphere/internal/provider/subscription/datasource"
	teamdatasource "github.com/computesphere/terraform-provider-computesphere/internal/provider/team/datasource"
)

// csProviderModel maps provider schema data to a Go type.
type csProviderModel struct {
	APIURL    types.String `tfsdk:"api_url"`
	APIToken  types.String `tfsdk:"api_token"`
	AccountID types.String `tfsdk:"account_id"`
}

// Ensure the implementation satisfies the expected interfaces.
var _ provider.Provider = &ComputeSphereProvider{}
var _ provider.ProviderWithFunctions = &ComputeSphereProvider{}

// ComputeSphereProvider is the provider implementation.
type ComputeSphereProvider struct {
	version   string
	Host      string
	APIToken  string
	AccountID string
	// HTTPClient, when set, overrides the default HTTP client used by both the
	// v1 and v2 API clients. Tests inject a go-vcr recorder client here so
	// cassette replay intercepts outbound requests instead of hitting the API.
	HTTPClient *http.Client
}

// ConfigFunc allows for flexible provider construction (for testing/extensibility)
type ConfigFunc func(p *ComputeSphereProvider)

func WithHost(host string) ConfigFunc {
	return func(p *ComputeSphereProvider) {
		p.Host = host
	}
}

func WithAPIToken(token string) ConfigFunc {
	return func(p *ComputeSphereProvider) {
		p.APIToken = token
	}
}

func WithAccountID(accountID string) ConfigFunc {
	return func(p *ComputeSphereProvider) {
		p.AccountID = accountID
	}
}

// WithHTTPClient overrides the HTTP client for the API client. Used by tests
// to inject a go-vcr recorder so cassettes can be recorded/replayed.
func WithHTTPClient(c *http.Client) ConfigFunc {
	return func(p *ComputeSphereProvider) {
		p.HTTPClient = c
	}
}

// New is a helper function to simplify provider server and testing implementation.
func New(version string, configFuncs ...ConfigFunc) func() provider.Provider {
	return func() provider.Provider {
		p := &ComputeSphereProvider{
			version: version,
		}
		for _, f := range configFuncs {
			f(p)
		}
		return p
	}
}

func (p *ComputeSphereProvider) Metadata(_ context.Context, _ provider.MetadataRequest, resp *provider.MetadataResponse) {
	resp.TypeName = "computesphere"
	resp.Version = p.version
}

var providerDescription = `The ComputeSphere provider is used to interact with and manage resources on ComputeSphere. The provider requires an API token and account ID to be used.`

func (p *ComputeSphereProvider) Schema(_ context.Context, _ provider.SchemaRequest, resp *provider.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: providerDescription,
		Attributes: map[string]schema.Attribute{
			"api_url": schema.StringAttribute{
				Optional:    true,
				Description: "The ComputeSphere API base URL. Defaults to `" + defaultAPIURL + "`. Values without a scheme or version path are normalized (e.g. `api.computesphere.com` becomes `" + defaultAPIURL + "`). Can also be set via the COMPUTESPHERE_API_URL environment variable. Only set this for non-default topologies.",
			},
			"api_token": schema.StringAttribute{
				Optional:    true,
				Sensitive:   true,
				Description: "The ComputeSphere API token. Can also be set via the COMPUTESPHERE_API_TOKEN environment variable.",
			},
			"account_id": schema.StringAttribute{
				Optional:    true,
				Description: "The ComputeSphere account ID. Can also be set via the COMPUTESPHERE_ACCOUNT_ID environment variable.",
			},
		},
	}
}

func (p *ComputeSphereProvider) Configure(ctx context.Context, req provider.ConfigureRequest, resp *provider.ConfigureResponse) {
	tflog.Info(ctx, "Configuring ComputeSphere Client")

	var config csProviderModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...) // single Get call
	if resp.Diagnostics.HasError() {
		return
	}

	// Unknown value checks
	if config.APIURL.IsUnknown() {
		resp.Diagnostics.AddAttributeError(
			path.Root("api_url"),
			"Unknown API URL",
			"The provider cannot create the ComputeSphere API client as there is an unknown configuration value for the API URL. Set the value statically in the configuration or use the COMPUTESPHERE_API_URL environment variable.",
		)
	}
	if config.APIToken.IsUnknown() {
		resp.Diagnostics.AddAttributeError(
			path.Root("api_token"),
			"Unknown API Token",
			"The provider cannot create the ComputeSphere API client as there is an unknown configuration value for the API token. Set the value statically in the configuration or use the COMPUTESPHERE_API_TOKEN environment variable.",
		)
	}
	if config.AccountID.IsUnknown() {
		resp.Diagnostics.AddAttributeError(
			path.Root("account_id"),
			"Unknown Account ID",
			"The provider cannot create the ComputeSphere API client as there is an unknown configuration value for the account ID. Set the value statically in the configuration or use the COMPUTESPHERE_ACCOUNT_ID environment variable.",
		)
	}
	if resp.Diagnostics.HasError() {
		return
	}

	// Set defaults to environment variables, but override with Terraform config value if set
	if p.Host == "" {
		p.Host = os.Getenv("COMPUTESPHERE_API_URL")
	}
	if p.APIToken == "" {
		p.APIToken = os.Getenv("COMPUTESPHERE_API_TOKEN")
	}
	if p.AccountID == "" {
		p.AccountID = os.Getenv("COMPUTESPHERE_ACCOUNT_ID")
	}
	if !config.APIURL.IsNull() {
		p.Host = config.APIURL.ValueString()
	}
	if !config.APIToken.IsNull() {
		p.APIToken = config.APIToken.ValueString()
	}
	if !config.AccountID.IsNull() {
		p.AccountID = config.AccountID.ValueString()
	}

	if p.APIToken == "" {
		resp.Diagnostics.AddAttributeError(
			path.Root("api_token"),
			"Missing ComputeSphere API Token",
			"The provider cannot create the ComputeSphere API client as there is a missing or empty value for the API token. Set the value in the configuration or use the COMPUTESPHERE_API_TOKEN environment variable.",
		)
	}
	if p.AccountID == "" {
		resp.Diagnostics.AddAttributeError(
			path.Root("account_id"),
			"Missing ComputeSphere Account ID",
			"The provider cannot create the ComputeSphere API client as there is a missing or empty value for the account ID. Set the value in the configuration or use the COMPUTESPHERE_ACCOUNT_ID environment variable.",
		)
	}
	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Debug(ctx, "Creating ComputeSphere API client")

	// Build the public v2 SDK client used by every resource and datasource.
	v2Base, urlErr := normalizeAPIURL(p.Host)
	if urlErr != nil {
		resp.Diagnostics.AddAttributeError(
			path.Root("api_url"),
			"Invalid ComputeSphere API URL",
			fmt.Sprintf(
				"The provider cannot create the ComputeSphere API client because the API URL is invalid: %s.\n\n"+
					"Expected a URL of the form %s. Set a valid value in the configuration or via the COMPUTESPHERE_API_URL environment variable, or omit it to use the default.",
				urlErr, defaultAPIURL,
			),
		)
		return
	}
	v2Opts := []csv2.ClientOption{
		csv2.WithRequestEditorFn(func(_ context.Context, req *http.Request) error {
			if p.APIToken != "" {
				req.Header.Set("Authorization", "Bearer "+p.APIToken)
			}
			// The API is account-scoped. Operations that take an explicit
			// account_id param send it themselves; account-scoped GET-by-id
			// operations rely on this header, so set it on every request.
			if p.AccountID != "" {
				req.Header.Set("x-account-id", p.AccountID)
			}
			return nil
		}),
	}
	if p.HTTPClient != nil {
		v2Opts = append(v2Opts, csv2.WithHTTPClient(p.HTTPClient))
	}
	v2Client, v2Err := csv2.NewClientWithResponses(v2Base, v2Opts...)
	if v2Err != nil {
		resp.Diagnostics.AddError(
			"Failed to initialize v2 API client",
			v2Err.Error(),
		)
		return
	}

	data := &cstypes.Data{
		V2Client:  v2Client,
		AccountID: p.AccountID,
	}
	resp.DataSourceData = data
	resp.ResourceData = data
}

// defaultAPIURL is the canonical base URL for the ComputeSphere v2 API.
const defaultAPIURL = "https://api.computesphere.com/v2"

// normalizeAPIURL resolves the raw api_url value (config attribute or
// COMPUTESPHERE_API_URL) into the v2 base URL the SDK client is built with.
// An empty value falls back to defaultAPIURL, a schemeless value (e.g.
// "api.computesphere.com") gets "https://" prepended, and an empty or legacy
// (/api/v1, /api/v2) path is rewritten to /v2 — the path the live API serves
// v2 on. Any other explicit path is preserved unchanged so consumers with
// unusual topologies can point directly at their own base path. Invalid or
// non-http(s) values return an error so misconfiguration surfaces at
// provider configure time instead of as a cryptic failure on first apply.
func normalizeAPIURL(raw string) (string, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return defaultAPIURL, nil
	}
	// Bare hosts like "api.computesphere.com" parse as a path, not a host;
	// assume HTTPS when no scheme was given.
	if !strings.Contains(raw, "://") {
		raw = "https://" + raw
	}
	u, err := url.Parse(raw)
	if err != nil {
		return "", fmt.Errorf("could not parse %q: %w", raw, err)
	}
	if u.Scheme != "http" && u.Scheme != "https" {
		return "", fmt.Errorf("unsupported scheme %q in %q, expected http or https", u.Scheme, raw)
	}
	if u.Host == "" {
		return "", fmt.Errorf("missing host in %q", raw)
	}
	switch strings.TrimSuffix(u.Path, "/") {
	case "", "/api/v1", "/api/v2":
		// Default and legacy version paths all map to the live v2 base path.
		u.Path = "/v2"
	default:
		u.Path = strings.TrimSuffix(u.Path, "/")
	}
	return u.String(), nil
}

func (p *ComputeSphereProvider) Resources(_ context.Context) []func() resource.Resource {
	return []func() resource.Resource{
		notificationresource.NewNotificationSettingResource,
		alertresource.NewAlertResource,
		deploymentresource.NewDeploymentResource,
		customdomainresource.NewCustomDomainResource,
		environmentresource.NewEnvironmentResource,
		guardrailresource.NewGuardrailResource,
		projectresource.NewProjectResource,
		serviceresource.NewServiceResource,
		apitokenresource.NewApiTokenResource,
		teamresource.NewTeamResource,
	}
}

func (p *ComputeSphereProvider) DataSources(_ context.Context) []func() datasource.DataSource {
	return []func() datasource.DataSource{
		notificationdatasource.NewNotificationSettingDataSource,
		alertdatasource.NewAlertDataSource,
		alertdatasource.NewAlertsDataSource,
		environmentdatasource.NewEnvironmentDataSource,
		environmentdatasource.NewEnvironmentsDataSource,
		environmentdatasource.NewEnvironmentVariablesDataSource,
		environmentdatasource.NewEnvironmentSecretsDataSource,
		guardraildatasource.NewGuardrailDataSource,
		plandatasource.NewPlanDataSource,
		plandatasource.NewPlansDataSource,
		projectdatasource.NewProjectDataSource,
		projectdatasource.NewProjectsDataSource,
		regiondatasource.NewRegionDataSource,
		regiondatasource.NewRegionsDataSource,
		servicedatasource.NewServiceDataSource,
		servicedatasource.NewServicesDataSource,
		subscriptiondatasource.NewSubscriptionDataSource,
		subscriptiondatasource.NewSubscriptionsDataSource,
		teamdatasource.NewTeamDataSource,
		teamdatasource.NewTeamsDataSource,
	}
}

func (p *ComputeSphereProvider) Functions(_ context.Context) []func() function.Function {
	return []func() function.Function{}
}
