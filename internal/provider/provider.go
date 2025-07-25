package provider

import (
	"context"
	"os"

	cs "github.com/computesphere/cli/cs"
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
	environmentresource "github.com/computesphere/terraform-provider-computesphere/internal/provider/environment/resource"
	guardrailresource "github.com/computesphere/terraform-provider-computesphere/internal/provider/guardrail/resource"
	notificationresource "github.com/computesphere/terraform-provider-computesphere/internal/provider/notification_setting/resource"
	planresource "github.com/computesphere/terraform-provider-computesphere/internal/provider/plan/resource"
	projectresource "github.com/computesphere/terraform-provider-computesphere/internal/provider/project/resource"
	serviceresource "github.com/computesphere/terraform-provider-computesphere/internal/provider/service/resource"
	subscriptionresource "github.com/computesphere/terraform-provider-computesphere/internal/provider/subscription/resource"
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
				Description: "The ComputeSphere API URL. Can also be set via the COMPUTESPHERE_API_URL environment variable.",
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
	conf := cs.NewConfiguration()
	if p.Host != "" {
		conf.Host = p.Host
	}
	if p.APIToken != "" {
		conf.XUserToken(p.APIToken)
	}
	if p.AccountID != "" {
		conf.XAccountID(p.AccountID)
	}
	client := cs.NewAPIClient(conf)
	data := &cstypes.Data{
		Client: client,
	}
	resp.DataSourceData = data
	resp.ResourceData = data
}

func (p *ComputeSphereProvider) Resources(_ context.Context) []func() resource.Resource {
	return []func() resource.Resource{
		notificationresource.NewNotificationSettingResource,
		alertresource.NewAlertResource,
		environmentresource.NewEnvironmentResource,
		guardrailresource.NewGuardrailResource,
		planresource.NewPlanResource,
		projectresource.NewProjectResource,
		serviceresource.NewServiceResource,
		apitokenresource.NewApiTokenResource,
		subscriptionresource.NewSubscriptionResource,
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
