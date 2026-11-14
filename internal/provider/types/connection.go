package types

import (
	"context"
	"net/http"
	"net/url"

	csv2 "github.com/computesphere/computesphere-go"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// Connection is a cache or queue connection read from
// GET /{caches|queues}/{id}/connection.
type Connection struct {
	Host     string  `json:"host"`
	Port     int64   `json:"port"`
	Username string  `json:"username"`
	Password string  `json:"password"`
	VHost    *string `json:"vhost"`
	URI      string  `json:"uri"`
	TLS      bool    `json:"tls"`
	Revealed bool    `json:"revealed"`
	CAFile   *string `json:"ca_file"`
}

// ConnectionModel holds a cache or queue connection's values; each data
// source maps it to its own model (cache_id or queue_id).
type ConnectionModel struct {
	ID       types.String `tfsdk:"-"`
	Reveal   types.Bool   `tfsdk:"-"`
	Host     types.String `tfsdk:"-"`
	Port     types.Int64  `tfsdk:"-"`
	Username types.String `tfsdk:"-"`
	Password types.String `tfsdk:"-"`
	URI      types.String `tfsdk:"-"`
	TLS      types.Bool   `tfsdk:"-"`
	Revealed types.Bool   `tfsdk:"-"`
	CAFile   types.String `tfsdk:"-"`
}

// ConnectionAttributes are the attributes both connection data sources share.
// noun is "cache" or "queue"; uriScheme its URI scheme.
func ConnectionAttributes(noun, uriScheme string) map[string]schema.Attribute {
	return map[string]schema.Attribute{
		noun + "_id": schema.StringAttribute{Required: true, Description: "The " + noun + "'s id."},
		"reveal": schema.BoolAttribute{
			Optional:    true,
			Description: "Include the password. Needs write access to the project. Default false: the password is masked.",
		},
		"host":     schema.StringAttribute{Computed: true, Description: "Private hostname, reachable only from services in the " + noun + "'s environment."},
		"port":     schema.Int64Attribute{Computed: true, Description: "TLS port."},
		"username": schema.StringAttribute{Computed: true, Description: "User name."},
		"password": schema.StringAttribute{Computed: true, Sensitive: true, Description: "Password; masked unless `reveal` is true."},
		"uri":      schema.StringAttribute{Computed: true, Sensitive: true, Description: "`" + uriScheme + "://` URI (TLS), with the password when `reveal` is true."},
		"tls":      schema.BoolAttribute{Computed: true, Description: "Always true: connect with TLS."},
		"revealed": schema.BoolAttribute{Computed: true, Description: "Whether the password is included."},
		"ca_file": schema.StringAttribute{
			Computed:    true,
			Description: "The file an app in the environment verifies the " + noun + "'s certificate with; null until it's ready.",
		},
	}
}

// ReadConnection reads a connection into m and returns it. collection is
// "caches" or "queues". It returns nil after adding a diagnostic.
func ReadConnection(ctx context.Context, client *csv2.ClientWithResponses, collection, noun string, m *ConnectionModel, resp *datasource.ReadResponse) *Connection {
	q := url.Values{}
	if m.Reveal.ValueBool() {
		q.Set("reveal", "true")
	}
	var c Connection
	status, raw, err := DoJSON(ctx, client, http.MethodGet, "/"+collection+"/"+url.PathEscape(m.ID.ValueString())+"/connection", q, nil, &c)
	if err != nil {
		resp.Diagnostics.AddError("Error reading "+noun+" connection", err.Error())
		return nil
	}
	if status == http.StatusNotFound {
		resp.Diagnostics.AddAttributeError(path.Root("id"), "No such "+noun, ProblemSummary(raw, status))
		return nil
	}
	if status != http.StatusOK {
		resp.Diagnostics.AddError("Error reading "+noun+" connection", ProblemSummary(raw, status))
		return nil
	}
	m.Host = types.StringValue(c.Host)
	m.Port = types.Int64Value(c.Port)
	m.Username = types.StringValue(c.Username)
	m.Password = types.StringValue(c.Password)
	m.URI = types.StringValue(c.URI)
	m.TLS = types.BoolValue(c.TLS)
	m.Revealed = types.BoolValue(c.Revealed)
	m.CAFile = types.StringPointerValue(c.CAFile)
	return &c
}
