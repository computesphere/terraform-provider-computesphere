package provider

import (
	"context"

	cs "github.com/computesphere/cli/cs"
	cstypes "github.com/computesphere/terraform-provider-computesphere/internal/provider/types"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

type ProjectDataSource struct {
	client *cs.APIClient
}

var _ datasource.DataSource = &ProjectDataSource{}

func NewProjectDataSource() datasource.DataSource {
	return &ProjectDataSource{}
}

func (d *ProjectDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = "computesphere_project"
}

func (d *ProjectDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = Schema(ctx)
}

type projectDataSourceModel struct {
	ID          types.String `tfsdk:"id"`
	Name        types.String `tfsdk:"name"`
	Description types.String `tfsdk:"description"`
	PlanName    types.String `tfsdk:"plan_name"`
	PlanValue   types.Int64  `tfsdk:"plan_value"`
	PlanID      types.String `tfsdk:"plan_id"`
	CreatedAt   types.String `tfsdk:"created_at"`
}

func (d *ProjectDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	data := cstypes.ConfigureDatasource(req, resp)
	if data != nil {
		d.client = data.Client
	}
}

func (d *ProjectDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var state projectDataSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	apiResp, httpResp, err := d.client.ProjectAPI.ProjectsIdGet(ctx, state.ID.ValueString()).Execute()
	if err != nil {
		if httpResp != nil && httpResp.StatusCode == 404 {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Error reading project", err.Error())
		return
	}
	if apiResp.Data == nil {
		resp.State.RemoveResource(ctx)
		return
	}
	project := apiResp.Data
	state.Name = types.StringValue(project.GetName())
	state.Description = types.StringValue(project.GetDescription())
	if project.Plan != nil {
		state.PlanName = types.StringPointerValue(project.Plan.Name)
		state.PlanID = types.StringPointerValue(project.Plan.Id)
		state.PlanValue = types.Int64Value(int64(project.GetPlanValue()))
	} else {
		state.PlanName = types.StringNull()
		state.PlanID = types.StringNull()
		state.PlanValue = types.Int64Null()
	}
	state.CreatedAt = types.StringValue(project.GetCreatedAt())
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}
