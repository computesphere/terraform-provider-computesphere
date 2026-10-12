package provider

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
)

func Schema(ctx context.Context) schema.Schema {
	keep := []planmodifier.String{stringplanmodifier.UseStateForUnknown()}
	replace := []planmodifier.String{stringplanmodifier.RequiresReplace()}
	return schema.Schema{
		Description: "Manages a SphereCache cache: a managed Valkey cache the platform runs for you. " +
			"Pick an instance type (see the `computesphere_cache_instance_types` data source) and an eviction " +
			"policy; the platform provisions it and generates its password. The password is never stored in " +
			"Terraform state.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed: true, PlanModifiers: keep,
				Description: "Cache identifier.",
			},
			"name": schema.StringAttribute{
				Required: true, PlanModifiers: replace,
				Description: "Lowercase letters, digits and hyphens, up to 40 characters; unique in the environment. " +
					"Changing it creates a new cache.",
			},
			"environment_id": schema.StringAttribute{
				Required: true, PlanModifiers: replace,
				Description: "Environment the cache belongs to. Only services in this environment can reach it. " +
					"Changing it creates a new cache.",
			},
			"engine": schema.StringAttribute{
				Optional: true, Computed: true, Default: stringdefault.StaticString("valkey"),
				PlanModifiers: replace,
				Description:   "Cache engine. Only `valkey` today.",
			},
			"version": schema.StringAttribute{
				Optional: true, Computed: true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown(), stringplanmodifier.RequiresReplace()},
				Description:   "Valkey version, e.g. `8.1`. Defaults to the latest supported. Changing it creates a new cache.",
			},
			"instance_type": schema.StringAttribute{
				Required: true,
				Description: "Instance type id, e.g. `sc-0.25c-1g`. List them with the `computesphere_cache_instance_types` " +
					"data source; an id ending in `-ha` adds a standby copy in another zone with automatic failover. " +
					"Changing it resizes the cache in place. On a smaller size, keys beyond its memory are evicted " +
					"(or writes are refused with `noeviction`).",
			},
			"eviction": schema.StringAttribute{
				Optional: true, Computed: true, PlanModifiers: keep,
				Description: "What happens when memory is full: `allkeys-lru` (the default) evicts the least recently " +
					"used keys; `noeviction` refuses writes until memory is freed. Changes apply in place.",
			},
			"high_availability": schema.BoolAttribute{
				Computed:    true,
				Description: "True when the cache has a standby copy in another zone (an `-ha` instance type).",
			},
			"status":     schema.StringAttribute{Computed: true, Description: "Current status, e.g. `available`."},
			"region":     schema.StringAttribute{Computed: true, PlanModifiers: keep, Description: "Region of the environment."},
			"project_id": schema.StringAttribute{Computed: true, PlanModifiers: keep, Description: "Project the cache belongs to."},
			"created_at": schema.StringAttribute{Computed: true, PlanModifiers: keep, Description: "Creation timestamp."},
		},
	}
}
