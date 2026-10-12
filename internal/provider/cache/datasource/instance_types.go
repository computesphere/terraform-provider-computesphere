package provider

import (
	"context"
	"net/http"
	"net/url"

	csv2 "github.com/computesphere/computesphere-go"
	cstypes "github.com/computesphere/terraform-provider-computesphere/internal/provider/types"
	"github.com/google/uuid"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// InstanceTypesDataSource is `computesphere_cache_instance_types`: the
// SphereCache size catalog with prices for an environment's region.
type InstanceTypesDataSource struct {
	client *csv2.ClientWithResponses
}

var _ datasource.DataSource = &InstanceTypesDataSource{}

func NewInstanceTypesDataSource() datasource.DataSource {
	return &InstanceTypesDataSource{}
}

func (d *InstanceTypesDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = "computesphere_cache_instance_types"
}

func (d *InstanceTypesDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Lists SphereCache instance types: memory, vCPU, whether the size is a high-availability twin, " +
			"and the monthly price. Pass `environment_id` to price for that environment's region and to get availability.",
		Attributes: map[string]schema.Attribute{
			"environment_id": schema.StringAttribute{
				Optional:    true,
				Description: "Price and check availability for this environment's region.",
			},
			"instance_types": schema.ListNestedAttribute{
				Computed:    true,
				Description: "The instance types, smallest first.",
				NestedObject: schema.NestedAttributeObject{Attributes: map[string]schema.Attribute{
					"id":                     schema.StringAttribute{Computed: true, Description: "Instance type id used on `computesphere_cache`, e.g. `sc-0.25c-1g`."},
					"name":                   schema.StringAttribute{Computed: true, Description: "Display name, e.g. `Cache 1 GB`."},
					"engine":                 schema.StringAttribute{Computed: true, Description: "Cache engine."},
					"vcpu":                   schema.Float64Attribute{Computed: true, Description: "vCPU; fractions are shared CPU. Null on a shared size."},
					"memory_mb":              schema.Int64Attribute{Computed: true, Description: "Memory for data, in MB."},
					"high_availability":      schema.BoolAttribute{Computed: true, Description: "True on a high-availability twin (an id ending in `-ha`): a standby copy in another zone with automatic failover."},
					"high_availability_type": schema.StringAttribute{Computed: true, Description: "The id of the size's high-availability twin; null when it has none."},
					"persistence":            schema.BoolAttribute{Computed: true, Description: "True when the cache keeps its data on disk across restarts."},
					"monthly_price_cents":    schema.Int64Attribute{Computed: true, Description: "Monthly price in USD cents for the region's price group; null until prices are published."},
					"price_group":            schema.StringAttribute{Computed: true, Description: "`standard` or `premium` (25% more)."},
					"available":              schema.BoolAttribute{Computed: true, Description: "False when the size can't be created here."},
					"unavailable_reason":     schema.StringAttribute{Computed: true, Description: "Why it's unavailable; null when available."},
				}},
			},
		},
	}
}

type apiPrice struct {
	Amount int64 `json:"amount"`
}

type apiInstanceType struct {
	ID                   string    `json:"id"`
	Name                 string    `json:"name"`
	Engine               string    `json:"engine"`
	VCPU                 *float64  `json:"vcpu"`
	MemoryMB             int64     `json:"memory_mb"`
	HighAvailability     bool      `json:"high_availability"`
	HighAvailabilityType *string   `json:"high_availability_type"`
	Persistence          bool      `json:"persistence"`
	MonthlyPrice         *apiPrice `json:"monthly_price"`
	PriceGroup           string    `json:"price_group"`
	Available            bool      `json:"available"`
	UnavailableReason    *string   `json:"unavailable_reason"`
}

type instanceTypeModel struct {
	ID                   types.String  `tfsdk:"id"`
	Name                 types.String  `tfsdk:"name"`
	Engine               types.String  `tfsdk:"engine"`
	VCPU                 types.Float64 `tfsdk:"vcpu"`
	MemoryMB             types.Int64   `tfsdk:"memory_mb"`
	HighAvailability     types.Bool    `tfsdk:"high_availability"`
	HighAvailabilityType types.String  `tfsdk:"high_availability_type"`
	Persistence          types.Bool    `tfsdk:"persistence"`
	MonthlyPriceCents    types.Int64   `tfsdk:"monthly_price_cents"`
	PriceGroup           types.String  `tfsdk:"price_group"`
	Available            types.Bool    `tfsdk:"available"`
	UnavailableReason    types.String  `tfsdk:"unavailable_reason"`
}

type instanceTypesModel struct {
	EnvironmentID types.String        `tfsdk:"environment_id"`
	InstanceTypes []instanceTypeModel `tfsdk:"instance_types"`
}

func (d *InstanceTypesDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	if data := cstypes.ConfigureDatasource(req, resp); data != nil {
		d.client = data.V2Client
	}
}

func (d *InstanceTypesDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var state instanceTypesModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	q := url.Values{}
	if v := state.EnvironmentID.ValueString(); v != "" {
		if _, err := uuid.Parse(v); err != nil {
			resp.Diagnostics.AddAttributeError(path.Root("environment_id"), "Invalid environment_id", err.Error())
			return
		}
		q.Set("environment_id", v)
	}

	var list struct {
		Items []apiInstanceType `json:"items"`
	}
	status, raw, err := cstypes.DoJSON(ctx, d.client, http.MethodGet, "/caches/instance-types", q, nil, &list)
	if err != nil {
		resp.Diagnostics.AddError("Error listing cache instance types", err.Error())
		return
	}
	if status != http.StatusOK {
		resp.Diagnostics.AddError("Error listing cache instance types", cstypes.ProblemSummary(raw, status))
		return
	}

	out := make([]instanceTypeModel, 0, len(list.Items))
	for _, t := range list.Items {
		m := instanceTypeModel{
			ID:                   types.StringValue(t.ID),
			Name:                 types.StringValue(t.Name),
			Engine:               types.StringValue(t.Engine),
			VCPU:                 types.Float64PointerValue(t.VCPU),
			MemoryMB:             types.Int64Value(t.MemoryMB),
			HighAvailability:     types.BoolValue(t.HighAvailability),
			HighAvailabilityType: types.StringPointerValue(t.HighAvailabilityType),
			Persistence:          types.BoolValue(t.Persistence),
			MonthlyPriceCents:    types.Int64Null(),
			PriceGroup:           types.StringValue(t.PriceGroup),
			Available:            types.BoolValue(t.Available),
			UnavailableReason:    types.StringPointerValue(t.UnavailableReason),
		}
		if t.MonthlyPrice != nil {
			m.MonthlyPriceCents = types.Int64Value(t.MonthlyPrice.Amount)
		}
		out = append(out, m)
	}
	state.InstanceTypes = out
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}
