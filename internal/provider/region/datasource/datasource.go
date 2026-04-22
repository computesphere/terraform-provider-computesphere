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

// RegionDataSource is the `computesphere_region` datasource. Stage 4e-scaffold
// migrated this from cli/cs.APIClient to the generated v2 SDK (sdk/go). It's
// the proof-of-pattern for the rest of the migration per RFC 0001 §16.
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
	Name types.String `tfsdk:"name"`
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

	found := false
	for _, r := range apiResp.JSON200.Items {
		if r.Name == state.Name.ValueString() {
			found = true
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
