package provider

import (
	"context"

	cs "github.com/computesphere/cli/cs"
	cstypes "github.com/computesphere/terraform-provider-computesphere/internal/provider/types"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

type GuardrailDataSource struct {
	client *cs.APIClient
}

var _ datasource.DataSource = &GuardrailDataSource{}

func NewGuardrailDataSource() datasource.DataSource {
	return &GuardrailDataSource{}
}

func (d *GuardrailDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = "computesphere_guardrail"
}

func (d *GuardrailDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = Schema(ctx)
}

type guardrailDataSourceModel struct {
	ID                   types.String              `tfsdk:"id"`
	Name                 types.String              `tfsdk:"name"`
	Description          types.String              `tfsdk:"description"`
	Effect               types.String              `tfsdk:"effect"`
	Message              types.String              `tfsdk:"message"`
	Rules                []map[string]types.String `tfsdk:"rules"`
	Scope                types.String              `tfsdk:"scope"`
	Status               types.Bool                `tfsdk:"status"`
	Type                 types.String              `tfsdk:"type"`
	AccountID            types.String              `tfsdk:"account_id"`
	CreatedBy            types.String              `tfsdk:"created_by"`
	IsPredefinedAssigned types.Bool                `tfsdk:"is_predefined_assigned"`
}

func (d *GuardrailDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	data := cstypes.ConfigureDatasource(req, resp)
	if data != nil {
		d.client = data.Client
	}
}

func (d *GuardrailDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var state guardrailDataSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	apiResp, httpResp, err := d.client.GuardrailsAPI.GuardrailsIdGet(ctx, state.ID.ValueString()).Execute()
	if err != nil {
		if httpResp != nil && httpResp.StatusCode == 404 {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Error reading guardrail", err.Error())
		return
	}
	if apiResp.Data == nil {
		resp.State.RemoveResource(ctx)
		return
	}
	g := apiResp.Data
	state.ID = types.StringValue(g.GetId())
	state.Name = types.StringValue(g.GetName())
	state.Description = types.StringValue(g.GetDescription())
	state.Effect = types.StringValue(g.GetEffect())
	state.Message = types.StringValue(g.GetMessage())
	state.Scope = types.StringValue(g.GetScope())
	state.Status = types.BoolValue(g.GetStatus())
	state.Type = types.StringValue(g.GetType())
	state.AccountID = types.StringValue(g.GetAccountId())
	state.CreatedBy = types.StringValue(g.GetCreatedBy())
	state.IsPredefinedAssigned = types.BoolValue(g.GetIsPredefinedAssigned())
	// Rules omitted for brevity; add as needed
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}
