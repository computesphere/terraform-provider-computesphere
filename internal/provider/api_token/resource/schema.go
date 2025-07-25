package provider

import (
	"context"

	shared "github.com/computesphere/terraform-provider-computesphere/internal/provider/types/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
)

func Schema(ctx context.Context) schema.Schema {
	return schema.Schema{
		Description: "Manages an API token resource.",
		Attributes: map[string]schema.Attribute{
			"id":          shared.ApiTokenID,
			"name":        shared.ApiTokenName,
			"scope":       shared.ApiTokenScope,
			"expiry":      shared.ApiTokenExpiry,
			"type":        shared.ApiTokenType,
			"account_id":  shared.ApiTokenAccountID,
			"account_ids": shared.ApiTokenAccountIDs,
			"project_ids": shared.ApiTokenProjectIDs,
			"token":       shared.ApiTokenToken,
			"created_at":  shared.ApiTokenCreatedAt,
			"user_id":     shared.ApiTokenUserID,
			"accounts":    shared.ApiTokenAccounts,
			"projects":    shared.ApiTokenProjects,
		},
	}
}
