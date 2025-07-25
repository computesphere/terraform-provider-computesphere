package provider

import (
	"context"

	cs "github.com/computesphere/cli/cs"
	cstypes "github.com/computesphere/terraform-provider-computesphere/internal/provider/types"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

type ProjectsDataSource struct {
	client *cs.APIClient
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
	PlanID      types.String `tfsdk:"plan_id"`
	CreatedAt   types.String `tfsdk:"created_at"`
}

type projectsDataSourceModel struct {
	Projects []projectItemModel `tfsdk:"projects"`
}

func (d *ProjectsDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	data := cstypes.ConfigureDatasource(req, resp)
	if data != nil {
		d.client = data.Client
	}
}

func (d *ProjectsDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var state projectsDataSourceModel
	apiResp, _, err := d.client.ProjectAPI.ProjectsGet(ctx).Execute()
	if err != nil {
		resp.Diagnostics.AddError("Error listing projects", err.Error())
		return
	}
	if apiResp.Data == nil {
		resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
		return
	}
	projects := make([]projectItemModel, 0, len(apiResp.Data))
	for _, p := range apiResp.Data {
		item := projectItemModel{
			ID:          types.StringValue(p.GetId()),
			Name:        types.StringValue(p.GetName()),
			Description: types.StringValue(p.GetDescription()),
			PlanID:      types.StringPointerValue(p.Plan.Id),
			CreatedAt:   types.StringValue(p.GetCreatedAt()),
		}
		projects = append(projects, item)
	}
	state.Projects = projects
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}
