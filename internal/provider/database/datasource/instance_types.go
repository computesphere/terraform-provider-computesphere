package provider

import (
	"context"
	"net/http"

	csv2 "github.com/computesphere/computesphere-go"
	cstypes "github.com/computesphere/terraform-provider-computesphere/internal/provider/types"
	"github.com/google/uuid"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// InstanceTypesDataSource is `computesphere_database_instance_types`: the
// SphereDB instance-type catalog with prices for an environment's region.
type InstanceTypesDataSource struct {
	client    *csv2.ClientWithResponses
	accountID string
}

var _ datasource.DataSource = &InstanceTypesDataSource{}

func NewInstanceTypesDataSource() datasource.DataSource {
	return &InstanceTypesDataSource{}
}

func (d *InstanceTypesDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = "computesphere_database_instance_types"
}

func (d *InstanceTypesDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Lists SphereDB instance types: Free, Flex, Standard and Plus sizes with vCPU, memory, " +
			"connection limit, included storage and monthly price. Pass `environment_id` to price for that " +
			"environment's region and to get availability.",
		Attributes: map[string]schema.Attribute{
			"environment_id": schema.StringAttribute{
				Optional:    true,
				Description: "Price and check availability for this environment's region.",
			},
			"engine": schema.StringAttribute{
				Optional:    true,
				Description: "Database engine. Only `postgresql` today.",
			},
			"instance_types": schema.ListNestedAttribute{
				Computed:    true,
				Description: "The instance types, smallest first.",
				NestedObject: schema.NestedAttributeObject{Attributes: map[string]schema.Attribute{
					"id":                        schema.StringAttribute{Computed: true, Description: "Instance type id used on `computesphere_database`, e.g. `pg-2c-4g`."},
					"name":                      schema.StringAttribute{Computed: true, Description: "Display name, e.g. `Standard 2`."},
					"family":                    schema.StringAttribute{Computed: true, Description: "`free`, `flex` (shared CPU), `standard` or `plus` (twice the memory per vCPU)."},
					"engine":                    schema.StringAttribute{Computed: true, Description: "Database engine."},
					"vcpu":                      schema.Float64Attribute{Computed: true, Description: "vCPU; fractions are shared CPU."},
					"memory_mb":                 schema.Int64Attribute{Computed: true, Description: "Memory in MB."},
					"burstable":                 schema.BoolAttribute{Computed: true, Description: "True on 1–2 vCPU types, which burst to their full vCPU."},
					"max_connections":           schema.Int64Attribute{Computed: true, Description: "Direct connections; pooled connections allow 5×."},
					"included_storage_gb":       schema.Int64Attribute{Computed: true, Description: "Storage included in the price."},
					"monthly_price_cents":       schema.Int64Attribute{Computed: true, Description: "Monthly price in USD cents for the region's price group."},
					"price_group":               schema.StringAttribute{Computed: true, Description: "`standard` or `premium` (25% more)."},
					"high_availability_allowed": schema.BoolAttribute{Computed: true, Description: "Whether `standby` and `three_zone` are offered."},
					"available":                 schema.BoolAttribute{Computed: true, Description: "False when the type can't be created in the environment's region."},
					"unavailable_reason":        schema.StringAttribute{Computed: true, Description: "Why it's unavailable; null when available."},
				}},
			},
		},
	}
}

type instanceTypeModel struct {
	ID                      types.String  `tfsdk:"id"`
	Name                    types.String  `tfsdk:"name"`
	Family                  types.String  `tfsdk:"family"`
	Engine                  types.String  `tfsdk:"engine"`
	VCPU                    types.Float64 `tfsdk:"vcpu"`
	MemoryMB                types.Int64   `tfsdk:"memory_mb"`
	Burstable               types.Bool    `tfsdk:"burstable"`
	MaxConnections          types.Int64   `tfsdk:"max_connections"`
	IncludedStorageGB       types.Int64   `tfsdk:"included_storage_gb"`
	MonthlyPriceCents       types.Int64   `tfsdk:"monthly_price_cents"`
	PriceGroup              types.String  `tfsdk:"price_group"`
	HighAvailabilityAllowed types.Bool    `tfsdk:"high_availability_allowed"`
	Available               types.Bool    `tfsdk:"available"`
	UnavailableReason       types.String  `tfsdk:"unavailable_reason"`
}

type instanceTypesModel struct {
	EnvironmentID types.String        `tfsdk:"environment_id"`
	Engine        types.String        `tfsdk:"engine"`
	InstanceTypes []instanceTypeModel `tfsdk:"instance_types"`
}

func (d *InstanceTypesDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	data := cstypes.ConfigureDatasource(req, resp)
	if data != nil {
		d.client = data.V2Client
		d.accountID = data.AccountID
	}
}

func (d *InstanceTypesDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var state instanceTypesModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	accountID, err := uuid.Parse(d.accountID)
	if err != nil {
		resp.Diagnostics.AddError("Invalid account_id", err.Error())
		return
	}
	params := &csv2.ListDatabaseInstanceTypesParams{XAccountId: accountID}
	if v := state.EnvironmentID.ValueString(); v != "" {
		envID, err := uuid.Parse(v)
		if err != nil {
			resp.Diagnostics.AddAttributeError(path.Root("environment_id"), "Invalid environment_id", err.Error())
			return
		}
		params.EnvironmentId = &envID
	}
	if v := state.Engine.ValueString(); v != "" {
		e := csv2.DatabaseEngine(v)
		params.Engine = &e
	}

	apiResp, err := d.client.ListDatabaseInstanceTypesWithResponse(ctx, params)
	if err != nil {
		resp.Diagnostics.AddError("Error listing database instance types", err.Error())
		return
	}
	if apiResp.StatusCode() != http.StatusOK || apiResp.JSON200 == nil {
		resp.Diagnostics.AddError("Error listing database instance types", cstypes.ProblemSummary(apiResp.Body, apiResp.StatusCode()))
		return
	}

	out := make([]instanceTypeModel, 0, len(apiResp.JSON200.Items))
	for _, t := range apiResp.JSON200.Items {
		m := instanceTypeModel{
			ID:                      types.StringValue(t.Id),
			Name:                    types.StringValue(t.Name),
			Family:                  types.StringValue(string(t.Family)),
			Engine:                  types.StringValue(string(t.Engine)),
			VCPU:                    types.Float64Value(float64(t.Vcpu)),
			MemoryMB:                types.Int64Value(int64(t.MemoryMb)),
			Burstable:               types.BoolValue(t.Burstable),
			MaxConnections:          types.Int64Value(int64(t.MaxConnections)),
			IncludedStorageGB:       types.Int64Value(int64(t.IncludedStorageGb)),
			MonthlyPriceCents:       types.Int64Value(t.MonthlyPrice.Amount),
			PriceGroup:              types.StringValue(string(t.PriceGroup)),
			HighAvailabilityAllowed: types.BoolValue(t.HighAvailabilityAllowed),
			Available:               types.BoolValue(t.Available),
			UnavailableReason:       types.StringNull(),
		}
		if t.UnavailableReason != nil {
			m.UnavailableReason = types.StringValue(string(*t.UnavailableReason))
		}
		out = append(out, m)
	}
	state.InstanceTypes = out
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}
