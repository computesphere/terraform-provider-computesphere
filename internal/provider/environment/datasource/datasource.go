package provider

import (
	"context"

	cs "github.com/computesphere/cli/cs"
	cstypes "github.com/computesphere/terraform-provider-computesphere/internal/provider/types"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

type EnvironmentDataSource struct {
	client *cs.APIClient
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
		d.client = data.Client
	}
}

func (d *EnvironmentDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var state environmentDataSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	apiResp, httpResp, err := d.client.EnvironmentAPI.EnvironmentsIdGet(ctx, state.ID.ValueString()).Execute()
	if err != nil {
		if httpResp != nil && httpResp.StatusCode == 404 {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Error reading environment", err.Error())
		return
	}
	if apiResp.Data == nil {
		resp.State.RemoveResource(ctx)
		return
	}
	env := apiResp.Data
	state.Name = types.StringValue(env.GetName())
	state.Region = types.StringValue(env.GetRegion())
	state.ProjectID = types.StringValue(env.GetProjectId())
	state.CreatedAt = types.StringValue(env.GetCreatedAt())
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}
