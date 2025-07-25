package provider

import (
	"context"

	cs "github.com/computesphere/cli/cs"
	cstypes "github.com/computesphere/terraform-provider-computesphere/internal/provider/types"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

type PlansDataSource struct {
	client *cs.APIClient
}

var _ datasource.DataSource = &PlansDataSource{}

func NewPlansDataSource() datasource.DataSource {
	return &PlansDataSource{}
}

func (d *PlansDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = "computesphere_plans"
}

func (d *PlansDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = PluralSchema(ctx)
}

type planItemModel struct {
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

type plansDataSourceModel struct {
	Plans []planItemModel `tfsdk:"plans"`
}

func (d *PlansDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	data := cstypes.ConfigureDatasource(req, resp)
	if data != nil {
		d.client = data.Client
	}
}

func (d *PlansDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var state plansDataSourceModel
	apiResp, _, err := d.client.PlanAPI.PlansGet(ctx).Execute()
	if err != nil {
		resp.Diagnostics.AddError("Error listing plans", err.Error())
		return
	}
	if apiResp.Data == nil {
		resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
		return
	}
	plans := make([]planItemModel, 0, len(apiResp.Data))
	for _, p := range apiResp.Data {
		item := planItemModel{
			ID:           types.StringValue(p.GetId()),
			Active:       types.BoolValue(p.GetActive()),
			Core:         types.Int64Value(int64(p.GetCore())),
			CountryCode:  types.StringValue(p.GetCountryCode()),
			CurrencyCode: types.StringValue(p.GetCurrencyCode()),
			Memory:       types.Int64Value(int64(p.GetMemory())),
			Name:         types.StringValue(p.GetName()),
			Price:        types.Float64Value(float64(p.GetPrice())),
			Slug:         types.StringValue(p.GetSlug()),
			Type:         types.StringValue(p.GetType()),
		}
		plans = append(plans, item)
	}
	state.Plans = plans
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}
