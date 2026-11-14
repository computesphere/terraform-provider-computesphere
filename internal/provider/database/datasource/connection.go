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
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// ConnectionDataSource reads how to connect to a SphereDB database: the
// pooled and direct routes, and the public ones when public access is on.
type ConnectionDataSource struct {
	client *csv2.ClientWithResponses
}

var _ datasource.DataSource = &ConnectionDataSource{}

func NewConnectionDataSource() datasource.DataSource {
	return &ConnectionDataSource{}
}

// apiRoute and apiConnection follow the GET /databases/{id}/connection body.
// They're decoded from the raw response: the SDK in use lacks some fields.
type apiRoute struct {
	Label    string `json:"label"`
	Host     string `json:"host"`
	Port     int64  `json:"port"`
	Database string `json:"database"`
	Username string `json:"username"`
	Password string `json:"password"`
	URI      string `json:"uri"`
}

type apiConnection struct {
	Engine                string    `json:"engine"`
	Pooled                *apiRoute `json:"pooled"`
	Direct                *apiRoute `json:"direct"`
	Revealed              bool      `json:"revealed"`
	Sslmode               string    `json:"sslmode"`
	CACertificate         *string   `json:"ca_certificate"`
	CACertificateFilename *string   `json:"ca_certificate_filename"`
	CAFile                *string   `json:"ca_file"`
	Public                *struct {
		Pooled                *apiRoute `json:"pooled"`
		Direct                *apiRoute `json:"direct"`
		CACertificate         *string   `json:"ca_certificate"`
		CACertificateFilename *string   `json:"ca_certificate_filename"`
	} `json:"public"`
}

var routeAttrTypes = map[string]attr.Type{
	"host": types.StringType, "port": types.Int64Type, "database": types.StringType,
	"username": types.StringType, "password": types.StringType, "uri": types.StringType,
}

func routeAttrs(desc string) schema.SingleNestedAttribute {
	return schema.SingleNestedAttribute{
		Computed: true, Description: desc,
		Attributes: map[string]schema.Attribute{
			"host":     schema.StringAttribute{Computed: true, Description: "Hostname."},
			"port":     schema.Int64Attribute{Computed: true, Description: "Port."},
			"database": schema.StringAttribute{Computed: true, Description: "Database name."},
			"username": schema.StringAttribute{Computed: true, Description: "User name."},
			"password": schema.StringAttribute{Computed: true, Sensitive: true, Description: "Password; masked unless `reveal` is true."},
			"uri":      schema.StringAttribute{Computed: true, Sensitive: true, Description: "Connection URI, with the password when `reveal` is true."},
		},
	}
}

func (d *ConnectionDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = "computesphere_database_connection"
}

func (d *ConnectionDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Reads how to connect to a SphereDB database: the pooled route (for your app) and the direct " +
			"route (for tools and migrations), and the public routes when public access is on. Private hosts are " +
			"reachable only from services in the database's environment. With `reveal = true` the password is " +
			"included, stored in Terraform state, and each read is recorded in the account's activity log.",
		Attributes: map[string]schema.Attribute{
			"database_id": schema.StringAttribute{Required: true, Description: "Database id."},
			"reveal": schema.BoolAttribute{
				Optional:    true,
				Description: "Include the password. Needs write access to the project. Default false: the password is masked.",
			},
			"engine":   schema.StringAttribute{Computed: true, Description: "Database engine, e.g. `postgresql`."},
			"sslmode":  schema.StringAttribute{Computed: true, Description: "TLS mode the private routes' URIs use."},
			"revealed": schema.BoolAttribute{Computed: true, Description: "Whether the password is included."},
			"ca_file": schema.StringAttribute{
				Computed:    true,
				Description: "Where an app in the environment finds the CA certificate for `sslmode=verify-full`; null when there is none.",
			},
			"ca_certificate": schema.StringAttribute{
				Computed: true, Description: "`documentdb` only: PEM CA certificates the private routes' URIs verify against.",
			},
			"ca_certificate_filename": schema.StringAttribute{
				Computed: true, Description: "`documentdb` only: the file name the URIs expect `ca_certificate` to be saved as.",
			},
			"pooled":        routeAttrs("Pooled route, for your app."),
			"direct":        routeAttrs("Direct route to the primary, for tools and migrations."),
			"public_pooled": routeAttrs("Pooled route on the public hostname; null unless public access is on and the pooled route is served there."),
			"public_direct": routeAttrs("Direct route on the public hostname; null unless public access is on."),
			"public_ca_certificate": schema.StringAttribute{
				Computed: true, Description: "PEM CA certificate the public routes' URIs verify against, when present; otherwise they use the system trust store.",
			},
			"public_ca_certificate_filename": schema.StringAttribute{
				Computed: true, Description: "The file name the public URIs expect `public_ca_certificate` to be saved as.",
			},
		},
	}
}

type connectionModel struct {
	DatabaseID                  types.String `tfsdk:"database_id"`
	Reveal                      types.Bool   `tfsdk:"reveal"`
	Engine                      types.String `tfsdk:"engine"`
	Sslmode                     types.String `tfsdk:"sslmode"`
	Revealed                    types.Bool   `tfsdk:"revealed"`
	CAFile                      types.String `tfsdk:"ca_file"`
	CACertificate               types.String `tfsdk:"ca_certificate"`
	CACertificateFilename       types.String `tfsdk:"ca_certificate_filename"`
	Pooled                      types.Object `tfsdk:"pooled"`
	Direct                      types.Object `tfsdk:"direct"`
	PublicPooled                types.Object `tfsdk:"public_pooled"`
	PublicDirect                types.Object `tfsdk:"public_direct"`
	PublicCACertificate         types.String `tfsdk:"public_ca_certificate"`
	PublicCACertificateFilename types.String `tfsdk:"public_ca_certificate_filename"`
}

func (d *ConnectionDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	if data := cstypes.ConfigureDatasource(req, resp); data != nil {
		d.client = data.V2Client
	}
}

func routeValue(r *apiRoute) types.Object {
	if r == nil {
		return types.ObjectNull(routeAttrTypes)
	}
	v, _ := types.ObjectValue(routeAttrTypes, map[string]attr.Value{
		"host": types.StringValue(r.Host), "port": types.Int64Value(r.Port), "database": types.StringValue(r.Database),
		"username": types.StringValue(r.Username), "password": types.StringValue(r.Password), "uri": types.StringValue(r.URI),
	})
	return v
}

func (d *ConnectionDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var m connectionModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &m)...)
	if resp.Diagnostics.HasError() {
		return
	}
	id, err := uuid.Parse(m.DatabaseID.ValueString())
	if err != nil {
		resp.Diagnostics.AddAttributeError(path.Root("database_id"), "Invalid database id", err.Error())
		return
	}
	reveal := m.Reveal.ValueBool()
	apiResp, err := d.client.GetDatabaseConnectionWithResponse(ctx, id, &csv2.GetDatabaseConnectionParams{Reveal: &reveal})
	if err != nil {
		resp.Diagnostics.AddError("Error reading database connection", err.Error())
		return
	}
	if apiResp.StatusCode() != http.StatusOK {
		resp.Diagnostics.AddError("Error reading database connection", cstypes.ProblemSummary(apiResp.Body, apiResp.StatusCode()))
		return
	}
	var c apiConnection
	if err := json.Unmarshal(apiResp.Body, &c); err != nil {
		resp.Diagnostics.AddError("Error reading database connection", "could not read the response: "+err.Error())
		return
	}
	m.Engine = types.StringValue(c.Engine)
	m.Sslmode = types.StringValue(c.Sslmode)
	m.Revealed = types.BoolValue(c.Revealed)
	m.CAFile = types.StringPointerValue(c.CAFile)
	m.CACertificate = types.StringPointerValue(c.CACertificate)
	m.CACertificateFilename = types.StringPointerValue(c.CACertificateFilename)
	m.Pooled = routeValue(c.Pooled)
	m.Direct = routeValue(c.Direct)
	m.PublicPooled = routeValue(nil)
	m.PublicDirect = routeValue(nil)
	m.PublicCACertificate = types.StringNull()
	m.PublicCACertificateFilename = types.StringNull()
	if c.Public != nil {
		m.PublicPooled = routeValue(c.Public.Pooled)
		m.PublicDirect = routeValue(c.Public.Direct)
		m.PublicCACertificate = types.StringPointerValue(c.Public.CACertificate)
		m.PublicCACertificateFilename = types.StringPointerValue(c.Public.CACertificateFilename)
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &m)...)
}
