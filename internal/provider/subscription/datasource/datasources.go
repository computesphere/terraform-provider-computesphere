package provider

import (
	"context"
	"net/http"

	csv2 "github.com/computesphere/computesphere-go"
	cstypes "github.com/computesphere/terraform-provider-computesphere/internal/provider/types"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

type SubscriptionsDataSource struct {
	client    *csv2.ClientWithResponses
	accountID string
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
		d.client = data.V2Client
		d.accountID = data.AccountID
	}
}

func (d *SubscriptionsDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var state subscriptionsDataSourceModel

	apiResp, err := d.client.ListSubscriptionsWithResponse(ctx, &csv2.ListSubscriptionsParams{})
	if err != nil {
		resp.Diagnostics.AddError("Error listing subscriptions", err.Error())
		return
	}
	if apiResp.StatusCode() != http.StatusOK || apiResp.JSON200 == nil {
		resp.Diagnostics.AddError("Error listing subscriptions", cstypes.ProblemSummary(apiResp.Body, apiResp.StatusCode()))
		return
	}

	subs := make([]subscriptionItemModel, 0, len(apiResp.JSON200.Items))
	for _, s := range apiResp.JSON200.Items {
		subs = append(subs, subscriptionItemModel{
			ID:           types.StringValue(s.Id),
			Name:         types.StringValue(s.Name),
			Active:       types.BoolValue(s.Active),
			CountryCode:  types.StringValue(s.CountryCode),
			CurrencyCode: types.StringValue(s.CurrencyCode),
			Price:        types.Float64Value(s.Price),
		})
	}
	state.Subscriptions = subs
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}
