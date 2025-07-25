package resource

import (
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var ApiTokenID = schema.StringAttribute{
	Computed:            true,
	Description:         "Unique identifier for the API token.",
	MarkdownDescription: "Unique identifier for the API token.",
}

var ApiTokenName = schema.StringAttribute{
	Required:            true,
	Description:         "Name of the API token.",
	MarkdownDescription: "Name of the API token.",
}

var ApiTokenScope = schema.StringAttribute{
	Required:            true,
	Description:         "Scope of the API token.",
	MarkdownDescription: "Scope of the API token.",
}

var ApiTokenExpiry = schema.StringAttribute{
	Optional:            true,
	Description:         "Expiry date of the API token.",
	MarkdownDescription: "Expiry date of the API token.",
}

var ApiTokenType = schema.StringAttribute{
	Optional:            true,
	Description:         "Type of the API token.",
	MarkdownDescription: "Type of the API token.",
}

var ApiTokenAccountID = schema.StringAttribute{
	Optional:            true,
	Description:         "Account ID associated with the API token.",
	MarkdownDescription: "Account ID associated with the API token.",
}

var ApiTokenAccountIDs = schema.ListAttribute{
	ElementType:         types.StringType,
	Optional:            true,
	Description:         "List of account IDs associated with the API token.",
	MarkdownDescription: "List of account IDs associated with the API token.",
}

var ApiTokenProjectIDs = schema.ListAttribute{
	ElementType:         types.StringType,
	Optional:            true,
	Description:         "List of project IDs associated with the API token.",
	MarkdownDescription: "List of project IDs associated with the API token.",
}

var ApiTokenToken = schema.StringAttribute{
	Computed:            true,
	Sensitive:           true,
	Description:         "The API token value.",
	MarkdownDescription: "The API token value.",
}

var ApiTokenCreatedAt = schema.StringAttribute{
	Computed:            true,
	Description:         "Creation timestamp of the API token.",
	MarkdownDescription: "Creation timestamp of the API token.",
}

var ApiTokenUserID = schema.StringAttribute{
	Computed:            true,
	Description:         "User ID associated with the API token.",
	MarkdownDescription: "User ID associated with the API token.",
}

var ApiTokenAccounts = schema.ListAttribute{
	ElementType:         types.StringType,
	Computed:            true,
	Description:         "List of account IDs associated with the API token.",
	MarkdownDescription: "List of account IDs associated with the API token.",
}

var ApiTokenProjects = schema.ListAttribute{
	ElementType:         types.StringType,
	Computed:            true,
	Description:         "List of project IDs associated with the API token.",
	MarkdownDescription: "List of project IDs associated with the API token.",
}
