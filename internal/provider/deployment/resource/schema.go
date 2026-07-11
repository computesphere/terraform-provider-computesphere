package provider

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

func Schema(ctx context.Context) schema.Schema {
	rr := []planmodifier.String{stringplanmodifier.RequiresReplace()}
	return schema.Schema{
		Description: "Manages a deployment: a running instance of a service in an environment. " +
			"Creating this resource creates the deployment, applies any env/secret vars, triggers a deploy, " +
			"and (by default) waits for it to become ready.",
		Attributes: map[string]schema.Attribute{
			"id":     schema.StringAttribute{Computed: true, Description: "Deployment id.", PlanModifiers: rr},
			"name":   schema.StringAttribute{Computed: true, Description: "Deployment name (inherited from the service)."},
			"status": schema.StringAttribute{Computed: true, Description: "Current deployment status."},
			"service_id": schema.StringAttribute{
				Required: true, Description: "Service this deployment runs.", PlanModifiers: rr,
			},
			"environment_id": schema.StringAttribute{
				Required: true, Description: "Target environment.", PlanModifiers: rr,
			},
			"project_id": schema.StringAttribute{Required: true, Description: "Owning project.", PlanModifiers: rr},
			"type": schema.StringAttribute{
				Required: true, Description: "Resource type (must match the service type, e.g. web-service).", PlanModifiers: rr,
			},
			"plan_id":      schema.StringAttribute{Optional: true, Description: "Compute tier override; defaults to the service's plan."},
			"image":        schema.StringAttribute{Optional: true, Description: "Container image reference (e.g. nginx:latest) for image-based deployments."},
			"port":         schema.Int64Attribute{Optional: true, Description: "Container port."},
			"sphere_count": schema.Int64Attribute{Optional: true, Description: "Number of spherelets (replicas) to run."},
			"env_vars": schema.MapAttribute{
				Optional: true, ElementType: types.StringType, Description: "Plain environment variables.",
			},
			"secret_vars": schema.MapAttribute{
				Optional: true, Sensitive: true, ElementType: types.StringType, Description: "Secret environment variables.",
			},
			"wait_for_ready": schema.BoolAttribute{
				Optional: true, Computed: true, Default: booldefault.StaticBool(true),
				Description: "Wait for the deployment to reach a ready state after deploying.",
			},
		},
	}
}
