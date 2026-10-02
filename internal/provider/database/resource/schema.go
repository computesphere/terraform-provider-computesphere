package provider

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
)

func Schema(ctx context.Context) schema.Schema {
	keep := []planmodifier.String{stringplanmodifier.UseStateForUnknown()}
	replace := []planmodifier.String{stringplanmodifier.RequiresReplace()}
	return schema.Schema{
		Description: "Manages a SphereDB database: a managed database the platform runs for you. " +
			"Pick an instance type (see the `computesphere_database_instance_types` data source), " +
			"storage and high availability; the platform provisions it and generates its credentials.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed: true, PlanModifiers: keep,
				Description: "Database identifier.",
			},
			"name": schema.StringAttribute{
				Required: true, PlanModifiers: replace,
				Description: "Lowercase letters, digits and hyphens; unique in the environment. Changing it creates a new database.",
			},
			"environment_id": schema.StringAttribute{
				Required: true, PlanModifiers: replace,
				Description: "Environment the database belongs to. Changing it creates a new database.",
			},
			"engine": schema.StringAttribute{
				Optional: true, Computed: true, Default: stringdefault.StaticString("postgresql"),
				PlanModifiers: replace,
				Description:   "Database engine. Only `postgresql` today.",
			},
			"version": schema.StringAttribute{
				Optional: true, Computed: true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown(), stringplanmodifier.RequiresReplace()},
				Description:   "Major version. Defaults to the latest supported. Changing it creates a new database.",
			},
			"instance_type": schema.StringAttribute{
				Optional: true, Computed: true, PlanModifiers: keep,
				Description: "Instance type id, e.g. `pg-2c-4g` (Standard 2). List them with the " +
					"`computesphere_database_instance_types` data source. Changing it resizes the database " +
					"in place with a short restart.",
			},
			"high_availability": schema.StringAttribute{
				Optional: true, Computed: true, PlanModifiers: keep,
				Description: "`none` (one instance), `standby` (a standby in another zone with automatic failover) or " +
					"`three_zone` (two standbys across three zones, synchronous). Standbys need an instance type " +
					"with 1 vCPU or more.",
			},
			"plan": schema.StringAttribute{
				Optional: true, Computed: true, PlanModifiers: keep,
				DeprecationMessage: "Use instance_type and high_availability instead.",
				Description: "Deprecated: use `instance_type` and `high_availability`. `hobby` is Standard 1, " +
					"`standard` is Standard 2 on zone-redundant storage, `ha` is Standard 2 with `three_zone`.",
			},
			"storage_gb": schema.Int64Attribute{
				Optional: true, Computed: true,
				PlanModifiers: []planmodifier.Int64{int64planmodifier.UseStateForUnknown()},
				Description:   "Storage in GB. Defaults to 10. It can only grow.",
			},
			"inject": schema.BoolAttribute{
				Optional: true, Computed: true, Default: booldefault.StaticBool(true),
				Description: "At create, give every service in the environment the pooled connection string as " +
					"`DATABASE_URL`. Only applies when the database is created.",
			},
			"status":        schema.StringAttribute{Computed: true, Description: "Current status, e.g. `available`."},
			"region":        schema.StringAttribute{Computed: true, PlanModifiers: keep, Description: "Region of the environment."},
			"project_id":    schema.StringAttribute{Computed: true, PlanModifiers: keep, Description: "Project the database belongs to."},
			"minor_version": schema.StringAttribute{Computed: true, Description: "Running minor version, e.g. `18.6`. Minor updates apply automatically."},
			"created_at":    schema.StringAttribute{Computed: true, PlanModifiers: keep, Description: "Creation timestamp."},
		},
	}
}
