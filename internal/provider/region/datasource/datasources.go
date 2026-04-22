package provider

import (
	"context"
	"net/http"

	csv2 "github.com/computesphere/computesphere-api/sdk/go"
	cstypes "github.com/computesphere/terraform-provider-computesphere/internal/provider/types"
	"github.com/google/uuid"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// RegionsDataSource is the `computesphere_regions` plural datasource —
// same Stage 4e migration as the singular sibling.
type RegionsDataSource struct {
	client    *csv2.ClientWithResponses
	accountID string
}

var _ datasource.DataSource = &RegionsDataSource{}

func NewRegionsDataSource() datasource.DataSource {
	return &RegionsDataSource{}
}

func (d *RegionsDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = "computesphere_regions"
}

func (d *RegionsDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = PluralSchema(ctx)
}

type regionsDataSourceModel struct {
	Regions []types.String `tfsdk:"regions"`
}

func (d *RegionsDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	data := cstypes.ConfigureDatasource(req, resp)
	if data != nil {
		d.client = data.V2Client
		d.accountID = data.AccountID
	}
}

func (d *RegionsDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var state regionsDataSourceModel

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

	// items is guaranteed non-nil by the v2 contract (RFC 0001 §6.1).
	regions := make([]types.String, 0, len(apiResp.JSON200.Items))
	for _, r := range apiResp.JSON200.Items {
		regions = append(regions, types.StringValue(r.Name))
	}
	state.Regions = regions
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}
