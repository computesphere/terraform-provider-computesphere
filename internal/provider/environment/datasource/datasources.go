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

type EnvironmentsDataSource struct {
	client    *csv2.ClientWithResponses
	accountID string
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
	ProjectID    types.String           `tfsdk:"project_id"`
	Environments []environmentItemModel `tfsdk:"environments"`
}

func (d *EnvironmentsDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	data := cstypes.ConfigureDatasource(req, resp)
	if data != nil {
		d.client = data.V2Client
		d.accountID = data.AccountID
	}
}

func (d *EnvironmentsDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var state environmentsDataSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	projectID, err := uuid.Parse(state.ProjectID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Invalid project_id", err.Error())
		return
	}

	apiResp, err := d.client.ListEnvironmentsWithResponse(ctx, &csv2.ListEnvironmentsParams{
		ProjectId: projectID,
	})
	if err != nil {
		resp.Diagnostics.AddError("Error listing environments", err.Error())
		return
	}
	if apiResp.StatusCode() != http.StatusOK || apiResp.JSON200 == nil {
		resp.Diagnostics.AddError("Error listing environments", cstypes.ProblemSummary(apiResp.Body, apiResp.StatusCode()))
		return
	}

	envs := make([]environmentItemModel, 0, len(apiResp.JSON200.Items))
	for _, e := range apiResp.JSON200.Items {
		envs = append(envs, environmentItemModel{
			ID:        types.StringValue(e.Id),
			Name:      types.StringValue(e.Name),
			Region:    types.StringValue(e.Region),
			ProjectID: types.StringValue(e.ProjectId.String()),
			CreatedAt: types.StringValue(e.CreatedAt.Format(time.RFC3339)),
		})
	}
	state.Environments = envs
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}
