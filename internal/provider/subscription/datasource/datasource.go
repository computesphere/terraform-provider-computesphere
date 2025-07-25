package provider

import (
	"context"

	cs "github.com/computesphere/cli/cs"
	cstypes "github.com/computesphere/terraform-provider-computesphere/internal/provider/types"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

type SubscriptionDataSource struct {
	client *cs.APIClient
}

var _ datasource.DataSource = &SubscriptionDataSource{}

func NewSubscriptionDataSource() datasource.DataSource {
	return &SubscriptionDataSource{}
}

func (d *SubscriptionDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = "computesphere_subscription"
}

func (d *SubscriptionDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = Schema(ctx)
}

type subscriptionDataSourceModel struct {
	ID           types.String  `tfsdk:"id"`
	Name         types.String  `tfsdk:"name"`
	Active       types.Bool    `tfsdk:"active"`
	CountryCode  types.String  `tfsdk:"country_code"`
	CurrencyCode types.String  `tfsdk:"currency_code"`
	Price        types.Float64 `tfsdk:"price"`
}

func (d *SubscriptionDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	data := cstypes.ConfigureDatasource(req, resp)
	if data != nil {
		d.client = data.Client
	}
}

func (d *SubscriptionDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var state subscriptionDataSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	apiResp, httpResp, err := d.client.SubscriptionAPI.SubscriptionsIdGet(ctx, state.ID.ValueString()).Execute()
	if err != nil {
		if httpResp != nil && httpResp.StatusCode == 404 {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Error reading subscription", err.Error())
		return
	}
	if apiResp.Data == nil {
		resp.State.RemoveResource(ctx)
		return
	}
	sub := apiResp.Data
	state.Name = types.StringValue(sub.GetName())
	state.Active = types.BoolValue(sub.GetActive())
	state.CountryCode = types.StringValue(sub.GetCountryCode())
	state.CurrencyCode = types.StringValue(sub.GetCurrencyCode())
	state.Price = types.Float64Value(float64(sub.GetPrice()))
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}
