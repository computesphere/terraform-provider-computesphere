package provider

import (
	"context"

	cs "github.com/computesphere/cli/cs"
	cstypes "github.com/computesphere/terraform-provider-computesphere/internal/provider/types"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

type PlanDataSource struct {
	client *cs.APIClient
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
		d.client = data.Client
	}
}

func (d *PlanDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var state planDataSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	apiResp, httpResp, err := d.client.PlanAPI.PlansIdGet(ctx, state.ID.ValueString()).Execute()
	if err != nil {
		if httpResp != nil && httpResp.StatusCode == 404 {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Error reading plan", err.Error())
		return
	}
	if apiResp.Data == nil {
		resp.State.RemoveResource(ctx)
		return
	}
	plan := apiResp.Data
	state.Active = types.BoolValue(plan.GetActive())
	state.Core = types.Int64Value(int64(plan.GetCore()))
	state.CountryCode = types.StringValue(plan.GetCountryCode())
	state.CurrencyCode = types.StringValue(plan.GetCurrencyCode())
	state.Memory = types.Int64Value(int64(plan.GetMemory()))
	state.Name = types.StringValue(plan.GetName())
	state.Price = types.Float64Value(float64(plan.GetPrice()))
	state.Slug = types.StringValue(plan.GetSlug())
	state.Type = types.StringValue(plan.GetType())
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}
