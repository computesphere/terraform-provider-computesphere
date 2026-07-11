package provider

import (
	"context"
	"net/http"

	csv2 "github.com/computesphere/computesphere-go"
	cstypes "github.com/computesphere/terraform-provider-computesphere/internal/provider/types"
	"github.com/google/uuid"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// EnvironmentSecretsDataSource exposes the secret environment variables for an
// environment. Values are marked sensitive.
type EnvironmentSecretsDataSource struct {
	client    *csv2.ClientWithResponses
	accountID string
}

var _ datasource.DataSource = &EnvironmentSecretsDataSource{}

func NewEnvironmentSecretsDataSource() datasource.DataSource {
	return &EnvironmentSecretsDataSource{}
}

func (d *EnvironmentSecretsDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = "computesphere_environment_secrets"
}

func (d *EnvironmentSecretsDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Secret environment variables for an environment.",
		Attributes: map[string]schema.Attribute{
			"environment_id": schema.StringAttribute{Required: true, Description: "Environment to read secrets for."},
			"secrets": schema.MapAttribute{
				Computed:    true,
				Sensitive:   true,
				ElementType: types.StringType,
				Description: "Map of secret name to value.",
			},
		},
	}
}

type environmentSecretsModel struct {
	EnvironmentID types.String `tfsdk:"environment_id"`
	Secrets       types.Map    `tfsdk:"secrets"`
}

func (d *EnvironmentSecretsDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	data := cstypes.ConfigureDatasource(req, resp)
	if data != nil {
		d.client = data.V2Client
		d.accountID = data.AccountID
	}
}

func (d *EnvironmentSecretsDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var state environmentSecretsModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	eid, err := uuid.Parse(state.EnvironmentID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Invalid environment_id", err.Error())
		return
	}

	apiResp, err := d.client.GetEnvironmentSecretsWithResponse(ctx, eid)
	if err != nil {
		resp.Diagnostics.AddError("Error reading environment secrets", err.Error())
		return
	}
	if apiResp.StatusCode() != http.StatusOK || apiResp.JSON200 == nil {
		resp.Diagnostics.AddError("Error reading environment secrets", cstypes.ProblemSummary(apiResp.Body, apiResp.StatusCode()))
		return
	}

	m, diags := types.MapValueFrom(ctx, types.StringType, apiResp.JSON200.Secrets)
	resp.Diagnostics.Append(diags...)
	state.Secrets = m
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}
