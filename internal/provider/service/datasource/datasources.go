package provider

import (
	"context"

	cs "github.com/computesphere/cli/cs"
	cstypes "github.com/computesphere/terraform-provider-computesphere/internal/provider/types"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

type ServicesDataSource struct {
	client *cs.APIClient
}

var _ datasource.DataSource = &ServicesDataSource{}

func NewServicesDataSource() datasource.DataSource {
	return &ServicesDataSource{}
}

func (d *ServicesDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = "computesphere_services"
}

func (d *ServicesDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = PluralSchema(ctx)
}

type serviceItemModel struct {
	ID           types.String `tfsdk:"id"`
	Name         types.String `tfsdk:"name"`
	ProjectID    types.String `tfsdk:"project_id"`
	Type         types.String `tfsdk:"type"`
	Active       types.Bool   `tfsdk:"active"`
	CreatedAt    types.String `tfsdk:"created_at"`
	LastOpenedAt types.String `tfsdk:"last_opened_at"`
}

type servicesDataSourceModel struct {
	Services []serviceItemModel `tfsdk:"services"`
}

func (d *ServicesDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	data := cstypes.ConfigureDatasource(req, resp)
	if data != nil {
		d.client = data.Client
	}
}

func (d *ServicesDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var state servicesDataSourceModel
	apiResp, _, err := d.client.ServiceAPI.ServicesGet(ctx).Execute()
	if err != nil {
		resp.Diagnostics.AddError("Error listing services", err.Error())
		return
	}
	if apiResp.Data == nil {
		resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
		return
	}
	services := make([]serviceItemModel, 0, len(apiResp.Data))
	for _, s := range apiResp.Data {
		item := serviceItemModel{
			ID:           types.StringValue(s.GetId()),
			Name:         types.StringValue(s.GetName()),
			ProjectID:    types.StringValue(s.GetProjectId()),
			Type:         types.StringValue(s.GetType()),
			Active:       types.BoolValue(s.GetActive()),
			CreatedAt:    types.StringValue(s.GetCreatedAt()),
			LastOpenedAt: types.StringValue(s.GetLastOpenedAt()),
		}
		services = append(services, item)
	}
	state.Services = services
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}
