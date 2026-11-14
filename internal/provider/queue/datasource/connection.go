package provider

import (
	"context"

	csv2 "github.com/computesphere/computesphere-go"
	cstypes "github.com/computesphere/terraform-provider-computesphere/internal/provider/types"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// ConnectionDataSource reads how to connect to a queue (broker).
type ConnectionDataSource struct {
	client *csv2.ClientWithResponses
}

var _ datasource.DataSource = &ConnectionDataSource{}

func NewConnectionDataSource() datasource.DataSource { return &ConnectionDataSource{} }

type queueConnectionModel struct {
	ID       types.String `tfsdk:"id"`
	Reveal   types.Bool   `tfsdk:"reveal"`
	Host     types.String `tfsdk:"host"`
	Port     types.Int64  `tfsdk:"port"`
	Username types.String `tfsdk:"username"`
	Password types.String `tfsdk:"password"`
	URI      types.String `tfsdk:"uri"`
	TLS      types.Bool   `tfsdk:"tls"`
	Revealed types.Bool   `tfsdk:"revealed"`
	CAFile   types.String `tfsdk:"ca_file"`
	VHost    types.String `tfsdk:"vhost"`
}

func (d *ConnectionDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = "computesphere_queue_connection"
}

func (d *ConnectionDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	attrs := cstypes.ConnectionAttributes("queue", "amqps")
	attrs["vhost"] = schema.StringAttribute{Computed: true, Description: "The virtual host, `/`."}
	resp.Schema = schema.Schema{
		Description: "Reads how to connect to a queue: host, port, user, virtual host and an `amqps://` URI over TLS. " +
			"The host is reachable only from services in the queue's environment. With `reveal = true` the password " +
			"is included and stored in Terraform state, and since every plan and apply reads the data source again, " +
			"each one reveals the password and is recorded in the account's audit log.",
		Attributes: attrs,
	}
}

func (d *ConnectionDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	if data := cstypes.ConfigureDatasource(req, resp); data != nil {
		d.client = data.V2Client
	}
}

func (d *ConnectionDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var q queueConnectionModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &q)...)
	if resp.Diagnostics.HasError() {
		return
	}
	m := cstypes.ConnectionModel{ID: q.ID, Reveal: q.Reveal}
	c := cstypes.ReadConnection(ctx, d.client, "queues", "queue", &m, resp)
	if c == nil {
		return
	}
	q.Host, q.Port, q.Username, q.Password, q.URI = m.Host, m.Port, m.Username, m.Password, m.URI
	q.TLS, q.Revealed, q.CAFile = m.TLS, m.Revealed, m.CAFile
	q.VHost = types.StringPointerValue(c.VHost)
	resp.Diagnostics.Append(resp.State.Set(ctx, &q)...)
}
