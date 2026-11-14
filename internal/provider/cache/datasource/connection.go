package provider

import (
	"context"

	csv2 "github.com/computesphere/computesphere-go"
	cstypes "github.com/computesphere/terraform-provider-computesphere/internal/provider/types"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
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

func (d *ConnectionDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var m cstypes.ConnectionModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &m)...)
	if resp.Diagnostics.HasError() {
		return
	}
	if cstypes.ReadConnection(ctx, d.client, "caches", "cache", &m, resp) == nil {
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &m)...)
}
