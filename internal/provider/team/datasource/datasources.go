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

type TeamsDataSource struct {
	client    *csv2.ClientWithResponses
	accountID string
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
	CreatedAt   types.String `tfsdk:"created_at"`
	UpdatedAt   types.String `tfsdk:"updated_at"`
}

type teamsDataSourceModel struct {
	Teams []teamItemModel `tfsdk:"teams"`
}

func (d *TeamsDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	data := cstypes.ConfigureDatasource(req, resp)
	if data != nil {
		d.client = data.V2Client
		d.accountID = data.AccountID
	}
}

func (d *TeamsDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var state teamsDataSourceModel

	accountID, err := uuid.Parse(d.accountID)
	if err != nil {
		resp.Diagnostics.AddError("Invalid account_id", err.Error())
		return
	}

	apiResp, err := d.client.ListTeamsWithResponse(ctx, &csv2.ListTeamsParams{
		AccountId: accountID,
	})
	if err != nil {
		resp.Diagnostics.AddError("Error listing teams", err.Error())
		return
	}
	if apiResp.StatusCode() != http.StatusOK || apiResp.JSON200 == nil {
		resp.Diagnostics.AddError("Error listing teams", cstypes.ProblemSummary(apiResp.Body, apiResp.StatusCode()))
		return
	}

	teams := make([]teamItemModel, 0, len(apiResp.JSON200.Items))
	for _, t := range apiResp.JSON200.Items {
		teams = append(teams, teamItemModel{
			ID:          types.StringValue(t.Id),
			Name:        types.StringValue(t.Name),
			AccountID:   types.StringValue(t.AccountId.String()),
			Description: types.StringPointerValue(t.Description),
			CreatedAt:   types.StringValue(t.CreatedAt.Format(time.RFC3339)),
			UpdatedAt:   types.StringValue(t.UpdatedAt.Format(time.RFC3339)),
		})
	}
	state.Teams = teams
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}
