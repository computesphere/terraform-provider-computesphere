package resource

import (
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
)

// Attribute definitions for the computesphere_custom_domain resource. A custom
// domain is an imperative sub-resource of a deployment (ADR 0132): one resource
// instance per hostname, so the resource is for_each-friendly. Only the
// server-issued id and hostname are exposed — the internal dom-<hash> slug is
// never surfaced in the schema or state.

var CustomDomainID = schema.StringAttribute{
	Computed:            true,
	Description:         "Server-issued identifier for the custom domain.",
	MarkdownDescription: "Server-issued identifier for the custom domain.",
}

var CustomDomainDeploymentID = schema.StringAttribute{
	Required:            true,
	Description:         "Identifier of the deployment this custom domain is bound to.",
	MarkdownDescription: "Identifier of the deployment this custom domain is bound to.",
}

var CustomDomainDomain = schema.StringAttribute{
	Required:            true,
	Description:         "Fully-qualified hostname to bind to the deployment.",
	MarkdownDescription: "Fully-qualified hostname to bind to the deployment.",
}

var CustomDomainHostname = schema.StringAttribute{
	Computed:            true,
	Description:         "Canonical fully-qualified hostname as reported by the server.",
	MarkdownDescription: "Canonical fully-qualified hostname as reported by the server.",
}

var CustomDomainStatus = schema.StringAttribute{
	Computed:            true,
	Description:         "Reconciliation state of the custom domain (pending_dns, validating, issuing, active, action_required).",
	MarkdownDescription: "Reconciliation state of the custom domain (`pending_dns`, `validating`, `issuing`, `active`, `action_required`).",
}

var CustomDomainVerified = schema.BoolAttribute{
	Computed:            true,
	Description:         "True once the custom domain is active (verified and serving).",
	MarkdownDescription: "True once the custom domain is active (verified and serving).",
}
