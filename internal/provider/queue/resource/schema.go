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
		Description: "Manages a SphereQueue broker: a dedicated RabbitMQ message broker the platform runs for you. " +
			"Pick an instance type (see the `computesphere_queue_instance_types` data source); the platform " +
			"provisions it and generates its user and password. The password is never stored in Terraform state.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed: true, PlanModifiers: keep,
				Description: "Broker identifier.",
			},
			"name": schema.StringAttribute{
				Required: true,
				Description: "Lowercase letters, digits and hyphens, up to 40 characters; unique in the environment. " +
					"It can't change after create; a change is refused at plan time.",
			},
			"environment_id": schema.StringAttribute{
				Required: true,
				Description: "Environment the broker belongs to. Only services in this environment can reach it. " +
					"It can't change after create; a change is refused at plan time.",
			},
			"engine": schema.StringAttribute{
				Optional: true, Computed: true, Default: stringdefault.StaticString("rabbitmq"),
				Description: "Broker engine. Only `rabbitmq` today. It can't change after create; a change is refused at plan time.",
			},
			"version": schema.StringAttribute{
				Optional: true, Computed: true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
				Description:   "RabbitMQ version, e.g. `4.3`. Defaults to the latest supported. It can't change after create; a change is refused at plan time.",
			},
			"instance_type": schema.StringAttribute{
				Required: true,
				Description: "Instance type id, e.g. `sq-1c-2g-s20g` (Queue 1). List them with the " +
					"`computesphere_queue_instance_types` data source. A high-availability instance type (an id ending in `-ha`) runs three instances across " +
					"zones with quorum queues. Changing it resizes the broker in place, with the same number of " +
					"instances; its disk grows with a larger size and never shrinks. With one instance, a queue " +
					"moves among Queue 1, Queue 2, Queue Plus 2 and Queue 4, and Queue 8 and Queue 16 can't change " +
					"size. High availability can't be turned on or off. The API refuses a change it can't make.",
			},
			"instances": schema.Int64Attribute{Computed: true, Description: "Instances in the broker: 1, or 3 on a high-availability instance type (an id ending in `-ha`)."},
			"storage_gb": schema.Int64Attribute{
				Computed:    true,
				Description: "Disk per instance, in GB. It grows with a larger size and never shrinks.",
			},
			"internal_host": schema.StringAttribute{
				Computed: true, PlanModifiers: keep,
				Description: "The queue's private host inside the environment. It never changes.",
			},
			"status":     schema.StringAttribute{Computed: true, Description: "Current status, e.g. `available`."},
			"region":     schema.StringAttribute{Computed: true, PlanModifiers: keep, Description: "Region of the environment."},
			"project_id": schema.StringAttribute{Computed: true, PlanModifiers: keep, Description: "Project the broker belongs to."},
			"created_at": schema.StringAttribute{Computed: true, PlanModifiers: keep, Description: "Creation timestamp."},
		},
	}
}
