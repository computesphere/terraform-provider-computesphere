package provider

import (
	"context"

	cs "github.com/computesphere/cli/cs"
	cstypes "github.com/computesphere/terraform-provider-computesphere/internal/provider/types"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

type SubscriptionsDataSource struct {
	client *cs.APIClient
}

var _ datasource.DataSource = &SubscriptionsDataSource{}

func NewSubscriptionsDataSource() datasource.DataSource {
	return &SubscriptionsDataSource{}
}

func (d *SubscriptionsDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = "computesphere_subscriptions"
}

func (d *SubscriptionsDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = PluralSchema(ctx)
}

type subscriptionItemModel struct {
	ID           types.String  `tfsdk:"id"`
	Name         types.String  `tfsdk:"name"`
	Active       types.Bool    `tfsdk:"active"`
	CountryCode  types.String  `tfsdk:"country_code"`
	CurrencyCode types.String  `tfsdk:"currency_code"`
	Price        types.Float64 `tfsdk:"price"`
}

type subscriptionsDataSourceModel struct {
	Subscriptions []subscriptionItemModel `tfsdk:"subscriptions"`
}

func (d *SubscriptionsDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	data := cstypes.ConfigureDatasource(req, resp)
	if data != nil {
		d.client = data.Client
	}
}

func (d *SubscriptionsDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var state subscriptionsDataSourceModel
	apiResp, _, err := d.client.SubscriptionAPI.SubscriptionsGet(ctx).Execute()
	if err != nil {
		resp.Diagnostics.AddError("Error listing subscriptions", err.Error())
		return
	}
	if apiResp.Data == nil {
		resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
		return
	}
	subs := make([]subscriptionItemModel, 0, len(apiResp.Data))
	for _, s := range apiResp.Data {
		item := subscriptionItemModel{
			ID:           types.StringValue(s.GetId()),
			Name:         types.StringValue(s.GetName()),
			Active:       types.BoolValue(s.GetActive()),
			CountryCode:  types.StringValue(s.GetCountryCode()),
			CurrencyCode: types.StringValue(s.GetCurrencyCode()),
			Price:        types.Float64Value(float64(s.GetPrice())),
		}
		subs = append(subs, item)
	}
	state.Subscriptions = subs
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}
