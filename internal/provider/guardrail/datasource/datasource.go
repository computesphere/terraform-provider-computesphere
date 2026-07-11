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

type GuardrailDataSource struct {
	client    *csv2.ClientWithResponses
	accountID string
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
	ID                   types.String `tfsdk:"id"`
	Name                 types.String `tfsdk:"name"`
	Description          types.String `tfsdk:"description"`
	Effect               types.String `tfsdk:"effect"`
	Message              types.String `tfsdk:"message"`
	Scope                types.String `tfsdk:"scope"`
	Status               types.Bool   `tfsdk:"status"`
	Type                 types.String `tfsdk:"type"`
	AccountID            types.String `tfsdk:"account_id"`
	CreatedBy            types.String `tfsdk:"created_by"`
	IsPredefinedAssigned types.Bool   `tfsdk:"is_predefined_assigned"`
}

func (d *GuardrailDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	data := cstypes.ConfigureDatasource(req, resp)
	if data != nil {
		d.client = data.V2Client
		d.accountID = data.AccountID
	}
}

func (m *guardrailDataSourceModel) apply(g *csv2.Guardrail) {
	m.ID = types.StringValue(g.Id.String())
	m.Name = types.StringValue(g.Name)
	m.Description = types.StringValue(g.Description)
	m.Effect = types.StringValue(string(g.Effect))
	m.Message = types.StringValue(g.Message)
	m.Scope = types.StringValue(string(g.Scope))
	m.Status = types.BoolValue(g.Status)
	m.Type = types.StringValue(string(g.Type))
	m.AccountID = types.StringValue(g.AccountId.String())
	m.CreatedBy = cstypes.UUIDPtrString(g.CreatedBy)
	m.IsPredefinedAssigned = types.BoolValue(g.IsPredefinedAssigned)
}

func (d *GuardrailDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var state guardrailDataSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	accountID, err := uuid.Parse(d.accountID)
	if err != nil {
		resp.Diagnostics.AddError("Invalid account_id", err.Error())
		return
	}
	gid, err := uuid.Parse(state.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Invalid guardrail id", err.Error())
		return
	}

	apiResp, err := d.client.GetGuardrailWithResponse(ctx, gid, &csv2.GetGuardrailParams{XAccountId: accountID})
	if err != nil {
		resp.Diagnostics.AddError("Error reading guardrail", err.Error())
		return
	}
	if apiResp.StatusCode() == http.StatusNotFound {
		resp.State.RemoveResource(ctx)
		return
	}
	if apiResp.StatusCode() != http.StatusOK || apiResp.JSON200 == nil {
		resp.Diagnostics.AddError("Error reading guardrail", cstypes.ProblemSummary(apiResp.Body, apiResp.StatusCode()))
		return
	}

	state.apply(apiResp.JSON200)
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}
