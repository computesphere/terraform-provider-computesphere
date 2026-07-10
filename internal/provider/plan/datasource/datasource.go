package provider

import (
	"context"
	"net/http"

	csv2 "github.com/computesphere/computesphere-go"
	cstypes "github.com/computesphere/terraform-provider-computesphere/internal/provider/types"
	"github.com/google/uuid"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

type PlanDataSource struct {
	client    *csv2.ClientWithResponses
	accountID string
}

var _ datasource.DataSource = &PlanDataSource{}

func NewPlanDataSource() datasource.DataSource {
	return &PlanDataSource{}
}

func (d *PlanDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = "computesphere_plan"
}

func (d *PlanDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = Schema(ctx)
}

type planDataSourceModel struct {
	ID           types.String  `tfsdk:"id"`
	Active       types.Bool    `tfsdk:"active"`
	Core         types.Int64   `tfsdk:"core"`
	CountryCode  types.String  `tfsdk:"country_code"`
	CurrencyCode types.String  `tfsdk:"currency_code"`
	Memory       types.Int64   `tfsdk:"memory"`
	Name         types.String  `tfsdk:"name"`
	Price        types.Float64 `tfsdk:"price"`
	Slug         types.String  `tfsdk:"slug"`
	Type         types.String  `tfsdk:"type"`
}

func (d *PlanDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	data := cstypes.ConfigureDatasource(req, resp)
	if data != nil {
		d.client = data.V2Client
		d.accountID = data.AccountID
	}
}

func (m *planDataSourceModel) apply(p *csv2.Plan) {
	m.ID = types.StringValue(p.Id)
	m.Active = types.BoolValue(p.Active)
	m.Core = types.Int64Value(int64(p.Core))
	m.CountryCode = types.StringValue(p.CountryCode)
	m.CurrencyCode = types.StringValue(p.CurrencyCode)
	m.Memory = types.Int64Value(int64(p.Memory))
	m.Name = types.StringValue(p.Name)
	m.Price = types.Float64Value(p.Price)
	m.Slug = types.StringValue(p.Slug)
	m.Type = types.StringValue(p.Type)
}

func (d *PlanDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var state planDataSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	pid, err := uuid.Parse(state.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Invalid plan id", err.Error())
		return
	}

	apiResp, err := d.client.GetPlanWithResponse(ctx, csv2.PlanId(pid))
	if err != nil {
		resp.Diagnostics.AddError("Error reading plan", err.Error())
		return
	}
	if apiResp.StatusCode() == http.StatusNotFound {
		resp.State.RemoveResource(ctx)
		return
	}
	if apiResp.StatusCode() != http.StatusOK || apiResp.JSON200 == nil {
		resp.Diagnostics.AddError("Error reading plan", cstypes.ProblemSummary(apiResp.Body, apiResp.StatusCode()))
		return
	}

	state.apply(apiResp.JSON200)
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}
