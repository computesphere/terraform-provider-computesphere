package provider

import (
	"context"
	"net/http"

	csv2 "github.com/computesphere/computesphere-go"
	cstypes "github.com/computesphere/terraform-provider-computesphere/internal/provider/types"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

type PlansDataSource struct {
	client    *csv2.ClientWithResponses
	accountID string
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
		d.client = data.V2Client
		d.accountID = data.AccountID
	}
}

func (d *PlansDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var state plansDataSourceModel

	apiResp, err := d.client.ListPlansWithResponse(ctx, &csv2.ListPlansParams{})
	if err != nil {
		resp.Diagnostics.AddError("Error listing plans", err.Error())
		return
	}
	if apiResp.StatusCode() != http.StatusOK || apiResp.JSON200 == nil {
		resp.Diagnostics.AddError("Error listing plans", cstypes.ProblemSummary(apiResp.Body, apiResp.StatusCode()))
		return
	}

	plans := make([]planItemModel, 0, len(apiResp.JSON200.Items))
	for _, p := range apiResp.JSON200.Items {
		plans = append(plans, planItemModel{
			ID:           types.StringValue(p.Id),
			Active:       types.BoolValue(p.Active),
			Core:         types.Int64Value(int64(p.Core)),
			CountryCode:  types.StringValue(p.CountryCode),
			CurrencyCode: types.StringValue(p.CurrencyCode),
			Memory:       types.Int64Value(int64(p.Memory)),
			Name:         types.StringValue(p.Name),
			Price:        types.Float64Value(p.Price),
			Slug:         types.StringValue(p.Slug),
			Type:         types.StringValue(p.Type),
		})
	}
	state.Plans = plans
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}
