package provider

import (
	"context"

	csv2 "github.com/computesphere/computesphere-go"
	cstypes "github.com/computesphere/terraform-provider-computesphere/internal/provider/types"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// ConnectionDataSource reads how to connect to a cache.
type ConnectionDataSource struct {
	client *csv2.ClientWithResponses
}

var _ datasource.DataSource = &ConnectionDataSource{}

func NewConnectionDataSource() datasource.DataSource { return &ConnectionDataSource{} }

func (d *ConnectionDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = "computesphere_cache_connection"
}

func (d *ConnectionDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Reads how to connect to a cache: host, port, user and a `rediss://` URI over TLS. The host is " +
			"reachable only from services in the cache's environment. With `reveal = true` the password is " +
			"included and stored in Terraform state, and since every plan and apply reads the data source again, " +
			"each one reveals the password and is recorded in the account's audit log.",
		Attributes: cstypes.ConnectionAttributes("cache", "rediss"),
	}
}

func (d *ConnectionDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	if data := cstypes.ConfigureDatasource(req, resp); data != nil {
		d.client = data.V2Client
	}
}

type cacheConnectionModel struct {
	CacheID  types.String `tfsdk:"cache_id"`
	Reveal   types.Bool   `tfsdk:"reveal"`
	Host     types.String `tfsdk:"host"`
	Port     types.Int64  `tfsdk:"port"`
	Username types.String `tfsdk:"username"`
	Password types.String `tfsdk:"password"`
	URI      types.String `tfsdk:"uri"`
	TLS      types.Bool   `tfsdk:"tls"`
	Revealed types.Bool   `tfsdk:"revealed"`
	CAFile   types.String `tfsdk:"ca_file"`
}

func (d *ConnectionDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var c cacheConnectionModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &c)...)
	if resp.Diagnostics.HasError() {
		return
	}
	m := cstypes.ConnectionModel{ID: c.CacheID, Reveal: c.Reveal}
	if cstypes.ReadConnection(ctx, d.client, "caches", "cache", &m, resp) == nil {
		return
	}
	c.Host, c.Port, c.Username, c.Password, c.URI = m.Host, m.Port, m.Username, m.Password, m.URI
	c.TLS, c.Revealed, c.CAFile = m.TLS, m.Revealed, m.CAFile
	resp.Diagnostics.Append(resp.State.Set(ctx, &c)...)
}
