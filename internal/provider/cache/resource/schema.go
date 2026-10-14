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
				Required: true,
				Description: "Lowercase letters, digits and hyphens, up to 40 characters; unique in the environment. " +
					"It can't change after create; a change is refused at plan time.",
			},
			"environment_id": schema.StringAttribute{
				Required: true,
				Description: "Environment the cache belongs to. Only services in this environment can reach it. " +
					"It can't change after create; a change is refused at plan time.",
			},
			"engine": schema.StringAttribute{
				Optional: true, Computed: true, Default: stringdefault.StaticString("valkey"),
				Description: "Cache engine. Only `valkey` today. It can't change after create; a change is refused at plan time.",
			},
			"version": schema.StringAttribute{
				Optional: true, Computed: true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
				Description:   "Valkey version, e.g. `8.1`. Defaults to the latest supported. It can't change after create; a change is refused at plan time.",
			},
			"instance_type": schema.StringAttribute{
				Required: true,
				Description: "Instance type id, e.g. `sc-0.25c-1g`. List them with the `computesphere_cache_instance_types` " +
					"data source. A high-availability instance type (an id ending in `-ha`) adds a standby copy in another zone with automatic failover. " +
					"Changing it resizes the cache in place. On a smaller size, keys beyond its memory are evicted " +
					"(or writes are refused with `noeviction`).",
			},
			"eviction": schema.StringAttribute{
				Optional: true, Computed: true, PlanModifiers: keep,
				Description: "What happens when memory is full: `allkeys-lru` (the default) evicts the least recently " +
					"used keys; `noeviction` refuses writes until memory is freed. It is set when the cache is created.",
			},
			"high_availability": schema.BoolAttribute{
				Computed:    true,
				Description: "True when the cache has high availability: a standby copy in another zone (a high-availability instance type, its id ending in `-ha`).",
			},
			"status":     schema.StringAttribute{Computed: true, Description: "Current status, e.g. `available`."},
			"region":     schema.StringAttribute{Computed: true, PlanModifiers: keep, Description: "Region of the environment."},
			"project_id": schema.StringAttribute{Computed: true, PlanModifiers: keep, Description: "Project the cache belongs to."},
			"created_at": schema.StringAttribute{Computed: true, PlanModifiers: keep, Description: "Creation timestamp."},
		},
	}
}
