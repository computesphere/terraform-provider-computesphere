package provider

import (
	"context"

	cs "github.com/computesphere/cli/cs"
	cstypes "github.com/computesphere/terraform-provider-computesphere/internal/provider/types"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

type RegionsDataSource struct {
	client *cs.APIClient
}

var _ datasource.DataSource = &RegionsDataSource{}

func NewRegionsDataSource() datasource.DataSource {
	return &RegionsDataSource{}
}

func (d *RegionsDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = "computesphere_regions"
}

func (d *RegionsDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = PluralSchema(ctx)
}

type regionsDataSourceModel struct {
	Regions []types.String `tfsdk:"regions"`
}

func (d *RegionsDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	data := cstypes.ConfigureDatasource(req, resp)
	if data != nil {
		d.client = data.Client
	}
}

func (d *RegionsDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var state regionsDataSourceModel
	apiResp, _, err := d.client.ClusterAPI.ClustersRegionsGet(ctx).Execute()
	if err != nil {
		resp.Diagnostics.AddError("Error listing regions", err.Error())
		return
	}
	if apiResp.Data == nil {
		resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
		return
	}
	regions := make([]types.String, 0, len(apiResp.Data))
	for _, r := range apiResp.Data {
		regions = append(regions, types.StringValue(r))
	}
	state.Regions = regions
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}
