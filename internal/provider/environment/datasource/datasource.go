package provider

import (
	"context"
	"net/http"
	"time"

	csv2 "github.com/computesphere/computesphere-go"
	cstypes "github.com/computesphere/terraform-provider-computesphere/internal/provider/types"
	"github.com/google/uuid"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

type EnvironmentDataSource struct {
	client    *csv2.ClientWithResponses
	accountID string
}

var _ datasource.DataSource = &EnvironmentDataSource{}

func NewEnvironmentDataSource() datasource.DataSource {
	return &EnvironmentDataSource{}
}

func (d *EnvironmentDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = "computesphere_environment"
}

func (d *EnvironmentDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = Schema(ctx)
}

type environmentDataSourceModel struct {
	ID        types.String `tfsdk:"id"`
	Name      types.String `tfsdk:"name"`
	Region    types.String `tfsdk:"region"`
	ProjectID types.String `tfsdk:"project_id"`
	CreatedAt types.String `tfsdk:"created_at"`
}

func (d *EnvironmentDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	data := cstypes.ConfigureDatasource(req, resp)
	if data != nil {
		d.client = data.V2Client
		d.accountID = data.AccountID
	}
}

func (d *EnvironmentDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var state environmentDataSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	eid, err := uuid.Parse(state.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Invalid environment id", err.Error())
		return
	}

	apiResp, err := d.client.GetEnvironmentWithResponse(ctx, eid)
	if err != nil {
		resp.Diagnostics.AddError("Error reading environment", err.Error())
		return
	}
	if apiResp.StatusCode() == http.StatusNotFound {
		resp.State.RemoveResource(ctx)
		return
	}
	if apiResp.StatusCode() != http.StatusOK || apiResp.JSON200 == nil {
		resp.Diagnostics.AddError("Error reading environment", cstypes.ProblemSummary(apiResp.Body, apiResp.StatusCode()))
		return
	}

	e := apiResp.JSON200
	state.Name = types.StringValue(e.Name)
	state.Region = types.StringValue(e.Region)
	state.ProjectID = types.StringValue(e.ProjectId.String())
	state.CreatedAt = types.StringValue(e.CreatedAt.Format(time.RFC3339))
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}
