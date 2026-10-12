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

// InstanceTypesDataSource is `computesphere_queue_instance_types`: the
// SphereQueue size catalog with prices and limits for an environment.
type InstanceTypesDataSource struct {
	client *csv2.ClientWithResponses
}

var _ datasource.DataSource = &InstanceTypesDataSource{}

func NewInstanceTypesDataSource() datasource.DataSource {
	return &InstanceTypesDataSource{}
}

func (d *InstanceTypesDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = "computesphere_queue_instance_types"
}

func (d *InstanceTypesDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Lists SphereQueue instance types: vCPU, memory, included disk, cluster size, limits and the " +
			"monthly price. Pass `environment_id` to get availability and prices for that environment.",
		Attributes: map[string]schema.Attribute{
			"environment_id": schema.StringAttribute{
				Optional:    true,
				Description: "Price and check availability for this environment.",
			},
			"instance_types": schema.ListNestedAttribute{
				Computed:    true,
				Description: "The instance types, smallest first.",
				NestedObject: schema.NestedAttributeObject{Attributes: map[string]schema.Attribute{
					"id":                     schema.StringAttribute{Computed: true, Description: "Instance type id used on `computesphere_queue`, e.g. `sq-1c-2g-s20g`."},
					"name":                   schema.StringAttribute{Computed: true, Description: "Display name, e.g. `Queue 1`."},
					"engine":                 schema.StringAttribute{Computed: true, Description: "Broker engine."},
					"family":                 schema.StringAttribute{Computed: true, Description: "`free` and `shared`: a virtual host on a shared broker. `dedicated`: a broker of your own."},
					"vcpu":                   schema.Float64Attribute{Computed: true, Description: "vCPU per member; null on shared types."},
					"memory_mb":              schema.Int64Attribute{Computed: true, Description: "Memory per member in MB; null on shared types."},
					"storage_gb":             schema.Int64Attribute{Computed: true, Description: "Disk included per member for queued messages; null on shared types."},
					"nodes":                  schema.Int64Attribute{Computed: true, Description: "Members of the broker's cluster: 1, or 3 on an `-ha` type (across three zones, with quorum queues)."},
					"high_availability":      schema.BoolAttribute{Computed: true, Description: "True on a three-member type."},
					"high_availability_type": schema.StringAttribute{Computed: true, Description: "The three-member twin of a one-member size; null when there is none."},
					"max_connections":        schema.Int64Attribute{Computed: true, Description: "Connection limit; 0 means no fixed limit."},
					"max_queues":             schema.Int64Attribute{Computed: true, Description: "Queue limit; 0 means no fixed limit."},
					"monthly_price_cents":    schema.Int64Attribute{Computed: true, Description: "Monthly price in USD cents for the region's price group; null until prices are published."},
					"price_group":            schema.StringAttribute{Computed: true, Description: "`standard` or `premium`."},
					"available":              schema.BoolAttribute{Computed: true, Description: "False when the type can't be created here."},
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
	Family               string    `json:"family"`
	VCPU                 *float64  `json:"vcpu"`
	MemoryMB             *int64    `json:"memory_mb"`
	StorageGB            *int64    `json:"storage_gb"`
	Nodes                int64     `json:"nodes"`
	HighAvailability     bool      `json:"high_availability"`
	HighAvailabilityType *string   `json:"high_availability_type"`
	MaxConnections       int64     `json:"max_connections"`
	MaxQueues            int64     `json:"max_queues"`
	MonthlyPrice         *apiPrice `json:"monthly_price"`
	PriceGroup           string    `json:"price_group"`
	Available            bool      `json:"available"`
	UnavailableReason    *string   `json:"unavailable_reason"`
}

type instanceTypeModel struct {
	ID                   types.String  `tfsdk:"id"`
	Name                 types.String  `tfsdk:"name"`
	Engine               types.String  `tfsdk:"engine"`
	Family               types.String  `tfsdk:"family"`
	VCPU                 types.Float64 `tfsdk:"vcpu"`
	MemoryMB             types.Int64   `tfsdk:"memory_mb"`
	StorageGB            types.Int64   `tfsdk:"storage_gb"`
	Nodes                types.Int64   `tfsdk:"nodes"`
	HighAvailability     types.Bool    `tfsdk:"high_availability"`
	HighAvailabilityType types.String  `tfsdk:"high_availability_type"`
	MaxConnections       types.Int64   `tfsdk:"max_connections"`
	MaxQueues            types.Int64   `tfsdk:"max_queues"`
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
	status, raw, err := cstypes.DoJSON(ctx, d.client, http.MethodGet, "/queues/instance-types", q, nil, &list)
	if err != nil {
		resp.Diagnostics.AddError("Error listing queue instance types", err.Error())
		return
	}
	if status != http.StatusOK {
		resp.Diagnostics.AddError("Error listing queue instance types", cstypes.ProblemSummary(raw, status))
		return
	}

	out := make([]instanceTypeModel, 0, len(list.Items))
	for _, t := range list.Items {
		m := instanceTypeModel{
			ID:                   types.StringValue(t.ID),
			Name:                 types.StringValue(t.Name),
			Engine:               types.StringValue(t.Engine),
			Family:               types.StringValue(t.Family),
			VCPU:                 types.Float64PointerValue(t.VCPU),
			MemoryMB:             types.Int64PointerValue(t.MemoryMB),
			StorageGB:            types.Int64PointerValue(t.StorageGB),
			Nodes:                types.Int64Value(t.Nodes),
			HighAvailability:     types.BoolValue(t.HighAvailability),
			HighAvailabilityType: types.StringPointerValue(t.HighAvailabilityType),
			MaxConnections:       types.Int64Value(t.MaxConnections),
			MaxQueues:            types.Int64Value(t.MaxQueues),
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
