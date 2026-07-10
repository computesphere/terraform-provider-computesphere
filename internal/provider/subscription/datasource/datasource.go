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

type SubscriptionDataSource struct {
	client    *csv2.ClientWithResponses
	accountID string
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
		d.client = data.V2Client
		d.accountID = data.AccountID
	}
}

func (d *SubscriptionDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var state subscriptionDataSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	sid, err := uuid.Parse(state.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Invalid subscription id", err.Error())
		return
	}

	apiResp, err := d.client.GetSubscriptionWithResponse(ctx, csv2.SubscriptionId(sid))
	if err != nil {
		resp.Diagnostics.AddError("Error reading subscription", err.Error())
		return
	}
	if apiResp.StatusCode() == http.StatusNotFound {
		resp.State.RemoveResource(ctx)
		return
	}
	if apiResp.StatusCode() != http.StatusOK || apiResp.JSON200 == nil {
		resp.Diagnostics.AddError("Error reading subscription", cstypes.ProblemSummary(apiResp.Body, apiResp.StatusCode()))
		return
	}

	s := apiResp.JSON200
	state.Name = types.StringValue(s.Name)
	state.Active = types.BoolValue(s.Active)
	state.CountryCode = types.StringValue(s.CountryCode)
	state.CurrencyCode = types.StringValue(s.CurrencyCode)
	state.Price = types.Float64Value(s.Price)
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}
