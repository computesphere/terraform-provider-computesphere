package provider

import (
	"context"

	cs "github.com/computesphere/cli/cs"
	cstypes "github.com/computesphere/terraform-provider-computesphere/internal/provider/types"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

type TeamsDataSource struct {
	client *cs.APIClient
}

var _ datasource.DataSource = &TeamsDataSource{}

func NewTeamsDataSource() datasource.DataSource {
	return &TeamsDataSource{}
}

func (d *TeamsDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = "computesphere_teams"
}

func (d *TeamsDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = PluralSchema(ctx)
}

type teamItemModel struct {
	ID          types.String `tfsdk:"id"`
	Name        types.String `tfsdk:"name"`
	AccountID   types.String `tfsdk:"account_id"`
	Description types.String `tfsdk:"description"`
}

type teamsDataSourceModel struct {
	Teams []teamItemModel `tfsdk:"teams"`
}

func (d *TeamsDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	data := cstypes.ConfigureDatasource(req, resp)
	if data != nil {
		d.client = data.Client
	}
}

func (d *TeamsDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var state teamsDataSourceModel
	// For future: If filtering by account_id is added, use the same pattern as the resource and single datasource.
	apiResp, _, err := d.client.TeamAPI.TeamsGet(ctx).Execute()
	if err != nil {
		resp.Diagnostics.AddError("Error listing teams", err.Error())
		return
	}
	if apiResp.Data == nil {
		resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
		return
	}
	teams := make([]teamItemModel, 0, len(apiResp.Data))
	for _, t := range apiResp.Data {
		item := teamItemModel{
			ID:          types.StringValue(t.GetId()),
			Name:        types.StringValue(t.GetName()),
			AccountID:   types.StringValue(t.GetAccountId()),
			Description: types.StringValue(t.GetDescription()),
		}
		teams = append(teams, item)
	}
	state.Teams = teams
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}
