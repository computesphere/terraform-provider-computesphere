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

// EnvironmentVariablesDataSource exposes the plain (non-secret) environment
// variables for an environment.
type EnvironmentVariablesDataSource struct {
	client    *csv2.ClientWithResponses
	accountID string
}

var _ datasource.DataSource = &EnvironmentVariablesDataSource{}

func NewEnvironmentVariablesDataSource() datasource.DataSource {
	return &EnvironmentVariablesDataSource{}
}

func (d *EnvironmentVariablesDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = "computesphere_environment_variables"
}

func (d *EnvironmentVariablesDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Plain (non-secret) environment variables for an environment.",
		Attributes: map[string]schema.Attribute{
			"environment_id": schema.StringAttribute{Required: true, Description: "Environment to read variables for."},
			"variables": schema.MapAttribute{
				Computed:    true,
				ElementType: types.StringType,
				Description: "Map of variable name to value.",
			},
		},
	}
}

type environmentVariablesModel struct {
	EnvironmentID types.String `tfsdk:"environment_id"`
	Variables     types.Map    `tfsdk:"variables"`
}

func (d *EnvironmentVariablesDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	data := cstypes.ConfigureDatasource(req, resp)
	if data != nil {
		d.client = data.V2Client
		d.accountID = data.AccountID
	}
}

func (d *EnvironmentVariablesDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var state environmentVariablesModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	eid, err := uuid.Parse(state.EnvironmentID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Invalid environment_id", err.Error())
		return
	}

	apiResp, err := d.client.GetEnvironmentVariablesWithResponse(ctx, eid)
	if err != nil {
		resp.Diagnostics.AddError("Error reading environment variables", err.Error())
		return
	}
	if apiResp.StatusCode() != http.StatusOK || apiResp.JSON200 == nil {
		resp.Diagnostics.AddError("Error reading environment variables", cstypes.ProblemSummary(apiResp.Body, apiResp.StatusCode()))
		return
	}

	m, diags := types.MapValueFrom(ctx, types.StringType, apiResp.JSON200.Variables)
	resp.Diagnostics.Append(diags...)
	state.Variables = m
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}
