package provider

import (
	"context"
	"encoding/json"
	"net/http"

	csv2 "github.com/computesphere/computesphere-go"
	cstypes "github.com/computesphere/terraform-provider-computesphere/internal/provider/types"
	"github.com/google/uuid"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// RegionDataSource is the `computesphere_region` datasource. It reads region
// metadata through the computesphere-go v2 client.
type RegionDataSource struct {
	client    *csv2.ClientWithResponses
	accountID string
}

var _ datasource.DataSource = &RegionDataSource{}

func NewRegionDataSource() datasource.DataSource {
	return &RegionDataSource{}
}

func (d *RegionDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = "computesphere_region"
}

func (d *RegionDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = Schema(ctx)
}

type regionDataSourceModel struct {
	Name             types.String `tfsdk:"name"`
	OutboundIPs      types.List   `tfsdk:"outbound_ips"`
	BuildOutboundIPs types.List   `tfsdk:"build_outbound_ips"`
}

// regionAddresses is one region of GET /v2/regions with the addresses its
// services and its builds connect out from, which computesphere-go v0.4.0
// does not model yet. Each is absent until the API publishes it.
type regionAddresses struct {
	Name             string   `json:"name"`
	OutboundIPs      []string `json:"outbound_ips"`
	BuildOutboundIPs []string `json:"build_outbound_ips"`
}

// addressList is a list attribute's value: null when the API publishes no
// addresses, so a configuration that needs them fails instead of using an
// empty allowlist.
func addressList(addresses []string) types.List {
	if len(addresses) == 0 {
		return types.ListNull(types.StringType)
	}
	values := make([]attr.Value, 0, len(addresses))
	for _, a := range addresses {
		values = append(values, types.StringValue(a))
	}
	return types.ListValueMust(types.StringType, values)
}

func (d *RegionDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	data := cstypes.ConfigureDatasource(req, resp)
	if data != nil {
		d.client = data.V2Client
		d.accountID = data.AccountID
	}
}

func (d *RegionDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var state regionDataSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	accountID, err := uuid.Parse(d.accountID)
	if err != nil {
		resp.Diagnostics.AddError("Invalid account_id", err.Error())
		return
	}

	apiResp, err := d.client.ListRegionsWithResponse(ctx, &csv2.ListRegionsParams{
		XAccountId: accountID,
	})
	if err != nil {
		resp.Diagnostics.AddError("Error listing regions", err.Error())
		return
	}
	if apiResp.StatusCode() != http.StatusOK || apiResp.JSON200 == nil {
		resp.Diagnostics.AddError(
			"Error listing regions",
			problemSummary(apiResp.Body, apiResp.StatusCode()),
		)
		return
	}

	var list struct {
		Items []regionAddresses `json:"items"`
	}
	if err := json.Unmarshal(apiResp.Body, &list); err != nil {
		resp.Diagnostics.AddError("Error reading regions", err.Error())
		return
	}
	found := false
	for _, r := range list.Items {
		if r.Name == state.Name.ValueString() {
			found = true
			state.OutboundIPs = addressList(r.OutboundIPs)
			state.BuildOutboundIPs = addressList(r.BuildOutboundIPs)
			break
		}
	}
	if !found {
		resp.State.RemoveResource(ctx)
		resp.Diagnostics.AddError("Region not found", state.Name.ValueString())
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

// problemSummary formats the RFC 7807 Problem body returned by the v2 API
// for Terraform diagnostic output. Prefers the raw body; when empty, falls
// back to the HTTP status text.
func problemSummary(body []byte, status int) string {
	if len(body) > 0 {
		return string(body)
	}
	return http.StatusText(status)
}
