package provider

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/listplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

func Schema(ctx context.Context) schema.Schema {
	return schema.Schema{
		Description: "Manages an API token resource. The token secret is only returned once, at creation.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:      true,
				Description:   "Unique identifier for the API token.",
				PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"name": schema.StringAttribute{
				Required:      true,
				Description:   "Name of the API token.",
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			"scope": schema.StringAttribute{
				Required:      true,
				Description:   "Scope of the API token (full, account, project).",
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			"expiry": schema.StringAttribute{
				Required:      true,
				Description:   "Expiry timestamp (RFC3339).",
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			"account_ids": schema.ListAttribute{
				ElementType:   types.StringType,
				Optional:      true,
				Computed:      true,
				Description:   "Account ids the token is scoped to.",
				PlanModifiers: []planmodifier.List{listplanmodifier.RequiresReplace()},
			},
			"project_ids": schema.ListAttribute{
				ElementType:   types.StringType,
				Optional:      true,
				Computed:      true,
				Description:   "Project ids the token is scoped to.",
				PlanModifiers: []planmodifier.List{listplanmodifier.RequiresReplace()},
			},
			"token": schema.StringAttribute{
				Computed:      true,
				Sensitive:     true,
				Description:   "The token secret. Only returned at creation.",
				PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"account_id": schema.StringAttribute{Computed: true, Description: "Owning account id."},
			"created_at": schema.StringAttribute{Computed: true, Description: "Creation timestamp."},
			"user_id":    schema.StringAttribute{Computed: true, Description: "Creator user id."},
		},
	}
}
