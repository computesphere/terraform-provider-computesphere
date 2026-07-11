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

type ServicesDataSource struct {
	client    *csv2.ClientWithResponses
	accountID string
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
	ProjectID types.String       `tfsdk:"project_id"`
	Services  []serviceItemModel `tfsdk:"services"`
}

func (d *ServicesDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	data := cstypes.ConfigureDatasource(req, resp)
	if data != nil {
		d.client = data.V2Client
		d.accountID = data.AccountID
	}
}

func (d *ServicesDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var state servicesDataSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	projectID, err := uuid.Parse(state.ProjectID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Invalid project_id", err.Error())
		return
	}

	apiResp, err := d.client.ListServicesWithResponse(ctx, &csv2.ListServicesParams{
		ProjectId: projectID,
	})
	if err != nil {
		resp.Diagnostics.AddError("Error listing services", err.Error())
		return
	}
	if apiResp.StatusCode() != http.StatusOK || apiResp.JSON200 == nil {
		resp.Diagnostics.AddError("Error listing services", cstypes.ProblemSummary(apiResp.Body, apiResp.StatusCode()))
		return
	}

	services := make([]serviceItemModel, 0, len(apiResp.JSON200.Items))
	for _, s := range apiResp.JSON200.Items {
		services = append(services, serviceItemModel{
			ID:           types.StringValue(s.Id),
			Name:         types.StringValue(s.Name),
			ProjectID:    types.StringValue(s.ProjectId.String()),
			Type:         types.StringValue(s.Type),
			Active:       types.BoolValue(s.Active),
			CreatedAt:    types.StringValue(s.CreatedAt.Format(time.RFC3339)),
			LastOpenedAt: cstypes.TimePtrString(s.LastOpenedAt),
		})
	}
	state.Services = services
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}
