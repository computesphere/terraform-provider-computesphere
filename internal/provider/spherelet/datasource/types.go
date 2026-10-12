package provider

import (
	"context"
	"net/http"

	csv2 "github.com/computesphere/computesphere-go"
	"github.com/computesphere/terraform-provider-computesphere/internal/provider/spherelet"
	cstypes "github.com/computesphere/terraform-provider-computesphere/internal/provider/types"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// TypesDataSource is `computesphere_spherelet_types`: the spherelet sizes a
// service or deployment can run on.
type TypesDataSource struct {
	client *csv2.ClientWithResponses
}

var _ datasource.DataSource = &TypesDataSource{}

func NewTypesDataSource() datasource.DataSource {
	return &TypesDataSource{}
}

func (d *TypesDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = "computesphere_spherelet_types"
}

func (d *TypesDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Lists the spherelet sizes a service or deployment can run on, with vCPU, memory " +
			"and price. Pass `region` to check availability in one region.",
		Attributes: map[string]schema.Attribute{
			"region": schema.StringAttribute{
				Optional:    true,
				Description: "Check availability in this region, e.g. `us-east-1`.",
			},
			"spherelet_types": schema.ListNestedAttribute{
				Computed:    true,
				Description: "The spherelet sizes.",
				NestedObject: schema.NestedAttributeObject{Attributes: map[string]schema.Attribute{
					"id":                  schema.StringAttribute{Computed: true, Description: "Size code to use as `spherelet_type`, e.g. `sl-1c-2g`."},
					"name":                schema.StringAttribute{Computed: true, Description: "Display name, e.g. `Standard 1`."},
					"family":              schema.StringAttribute{Computed: true, Description: "Size family: `flex`, `standard` or `plus`."},
					"vcpu":                schema.Float64Attribute{Computed: true, Description: "vCPU; fractions are shared CPU."},
					"cpu_guaranteed_vcpu": schema.Float64Attribute{Computed: true, Description: "vCPU guaranteed at all times. Up to `vcpu` is used when spare CPU is available."},
					"memory_mb":           schema.Int64Attribute{Computed: true, Description: "Memory in MB."},
					"monthly_price_usd":   schema.Float64Attribute{Computed: true, Description: "Monthly price in USD."},
					"annual_price_usd":    schema.Float64Attribute{Computed: true, Description: "Annual price in USD."},
					"available":           schema.BoolAttribute{Computed: true, Description: "Whether the size can be ordered now (in `region`, when set)."},
					"aliases": schema.ListAttribute{
						Computed: true, ElementType: types.StringType,
						Description: "Other names accepted as `spherelet_type` for this size, e.g. `MAX`.",
					},
				}},
			},
		},
	}
}

type typeModel struct {
	ID                types.String  `tfsdk:"id"`
	Name              types.String  `tfsdk:"name"`
	Family            types.String  `tfsdk:"family"`
	VCPU              types.Float64 `tfsdk:"vcpu"`
	CPUGuaranteedVCPU types.Float64 `tfsdk:"cpu_guaranteed_vcpu"`
	MemoryMB          types.Int64   `tfsdk:"memory_mb"`
	MonthlyPriceUSD   types.Float64 `tfsdk:"monthly_price_usd"`
	AnnualPriceUSD    types.Float64 `tfsdk:"annual_price_usd"`
	Available         types.Bool    `tfsdk:"available"`
	Aliases           types.List    `tfsdk:"aliases"`
}

type typesModel struct {
	Region         types.String `tfsdk:"region"`
	SphereletTypes []typeModel  `tfsdk:"spherelet_types"`
}

func (d *TypesDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	data := cstypes.ConfigureDatasource(req, resp)
	if data != nil {
		d.client = data.V2Client
	}
}

func (d *TypesDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var state typesModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	items, status, body, err := spherelet.List(ctx, d.client, state.Region.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Error listing spherelet sizes", err.Error())
		return
	}
	if status != http.StatusOK {
		resp.Diagnostics.AddError("Error listing spherelet sizes", cstypes.ProblemSummary(body, status))
		return
	}

	out := make([]typeModel, 0, len(items))
	for _, t := range items {
		aliases := t.Aliases
		if aliases == nil {
			aliases = []string{}
		}
		al, diags := types.ListValueFrom(ctx, types.StringType, aliases)
		resp.Diagnostics.Append(diags...)
		if resp.Diagnostics.HasError() {
			return
		}
		out = append(out, typeModel{
			ID:                types.StringValue(t.ID),
			Name:              types.StringValue(t.Name),
			Family:            types.StringValue(t.Family),
			VCPU:              types.Float64Value(t.VCPU),
			CPUGuaranteedVCPU: types.Float64Value(t.CPUGuaranteedVCPU),
			MemoryMB:          types.Int64Value(t.MemoryMB),
			MonthlyPriceUSD:   types.Float64Value(t.MonthlyPriceUSD),
			AnnualPriceUSD:    types.Float64Value(t.AnnualPriceUSD),
			Available:         types.BoolValue(t.Available),
			Aliases:           al,
		})
	}
	state.SphereletTypes = out
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}
