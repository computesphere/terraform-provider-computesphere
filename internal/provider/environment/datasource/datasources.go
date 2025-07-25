package provider

import (
	"context"

	cs "github.com/computesphere/cli/cs"
	cstypes "github.com/computesphere/terraform-provider-computesphere/internal/provider/types"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

type EnvironmentsDataSource struct {
	client *cs.APIClient
}

var _ datasource.DataSource = &EnvironmentsDataSource{}

func NewEnvironmentsDataSource() datasource.DataSource {
	return &EnvironmentsDataSource{}
}

func (d *EnvironmentsDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = "computesphere_environments"
}

func (d *EnvironmentsDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = PluralSchema(ctx)
}

type environmentItemModel struct {
	ID        types.String `tfsdk:"id"`
	Name      types.String `tfsdk:"name"`
	Region    types.String `tfsdk:"region"`
	ProjectID types.String `tfsdk:"project_id"`
	CreatedAt types.String `tfsdk:"created_at"`
}

type environmentsDataSourceModel struct {
	Environments []environmentItemModel `tfsdk:"environments"`
}

func (d *EnvironmentsDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	data := cstypes.ConfigureDatasource(req, resp)
	if data != nil {
		d.client = data.Client
	}
}

func (d *EnvironmentsDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var state environmentsDataSourceModel
	apiResp, _, err := d.client.EnvironmentAPI.EnvironmentsGet(ctx).Execute()
	if err != nil {
		resp.Diagnostics.AddError("Error listing environments", err.Error())
		return
	}
	if apiResp.Data == nil {
		resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
		return
	}
	envs := make([]environmentItemModel, 0, len(apiResp.Data))
	for _, e := range apiResp.Data {
		item := environmentItemModel{
			ID:        types.StringValue(e.GetId()),
			Name:      types.StringValue(e.GetName()),
			Region:    types.StringValue(e.GetRegion()),
			ProjectID: types.StringValue(e.GetProjectId()),
			CreatedAt: types.StringValue(e.GetCreatedAt()),
		}
		envs = append(envs, item)
	}
	state.Environments = envs
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}
