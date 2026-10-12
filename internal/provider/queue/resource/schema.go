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
		Description: "Manages a SphereQueue broker: a dedicated RabbitMQ message broker the platform runs for you. " +
			"Pick an instance type (see the `computesphere_queue_instance_types` data source); the platform " +
			"provisions it and generates its user and password. The password is never stored in Terraform state.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed: true, PlanModifiers: keep,
				Description: "Broker identifier.",
			},
			"name": schema.StringAttribute{
				Required: true, PlanModifiers: replace,
				Description: "Lowercase letters, digits and hyphens, up to 40 characters; unique in the environment. " +
					"Changing it creates a new broker.",
			},
			"environment_id": schema.StringAttribute{
				Required: true, PlanModifiers: replace,
				Description: "Environment the broker belongs to. Only services in this environment can reach it. " +
					"Changing it creates a new broker.",
			},
			"engine": schema.StringAttribute{
				Optional: true, Computed: true, Default: stringdefault.StaticString("rabbitmq"),
				PlanModifiers: replace,
				Description:   "Broker engine. Only `rabbitmq` today.",
			},
			"version": schema.StringAttribute{
				Optional: true, Computed: true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown(), stringplanmodifier.RequiresReplace()},
				Description:   "RabbitMQ version, e.g. `4.3`. Defaults to the latest supported. Changing it creates a new broker.",
			},
			"instance_type": schema.StringAttribute{
				Required: true,
				Description: "Instance type id, e.g. `sq-1c-2g-s20g` (Queue 1). List them with the " +
					"`computesphere_queue_instance_types` data source; an id ending in `-ha` runs three members across " +
					"zones with quorum queues. Changing it resizes the broker in place; a size with less disk than " +
					"the current one is refused.",
			},
			"nodes":      schema.Int64Attribute{Computed: true, Description: "Members of the broker's cluster: 1, or 3 on an `-ha` instance type."},
			"status":     schema.StringAttribute{Computed: true, Description: "Current status, e.g. `available`."},
			"region":     schema.StringAttribute{Computed: true, PlanModifiers: keep, Description: "Region of the environment."},
			"project_id": schema.StringAttribute{Computed: true, PlanModifiers: keep, Description: "Project the broker belongs to."},
			"created_at": schema.StringAttribute{Computed: true, PlanModifiers: keep, Description: "Creation timestamp."},
		},
	}
}
