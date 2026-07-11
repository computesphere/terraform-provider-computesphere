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

type ProjectsDataSource struct {
	client    *csv2.ClientWithResponses
	accountID string
}

var _ datasource.DataSource = &ProjectsDataSource{}

func NewProjectsDataSource() datasource.DataSource {
	return &ProjectsDataSource{}
}

func (d *ProjectsDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = "computesphere_projects"
}

func (d *ProjectsDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = PluralSchema(ctx)
}

type projectItemModel struct {
	ID          types.String `tfsdk:"id"`
	Name        types.String `tfsdk:"name"`
	Description types.String `tfsdk:"description"`
	CreatedAt   types.String `tfsdk:"created_at"`
}

type projectsDataSourceModel struct {
	Projects []projectItemModel `tfsdk:"projects"`
}

func (d *ProjectsDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	data := cstypes.ConfigureDatasource(req, resp)
	if data != nil {
		d.client = data.V2Client
		d.accountID = data.AccountID
	}
}

func (d *ProjectsDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var state projectsDataSourceModel

	accountID, err := uuid.Parse(d.accountID)
	if err != nil {
		resp.Diagnostics.AddError("Invalid account_id", err.Error())
		return
	}

	apiResp, err := d.client.ListProjectsWithResponse(ctx, &csv2.ListProjectsParams{
		AccountId: accountID,
	})
	if err != nil {
		resp.Diagnostics.AddError("Error listing projects", err.Error())
		return
	}
	if apiResp.StatusCode() != http.StatusOK || apiResp.JSON200 == nil {
		resp.Diagnostics.AddError("Error listing projects", cstypes.ProblemSummary(apiResp.Body, apiResp.StatusCode()))
		return
	}

	projects := make([]projectItemModel, 0, len(apiResp.JSON200.Items))
	for _, p := range apiResp.JSON200.Items {
		projects = append(projects, projectItemModel{
			ID:          types.StringValue(p.Id),
			Name:        types.StringValue(p.Name),
			Description: types.StringPointerValue(p.Description),
			CreatedAt:   types.StringValue(p.CreatedAt.Format(time.RFC3339)),
		})
	}
	state.Projects = projects
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}
