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
			"plan_id": schema.StringAttribute{Optional: true, Description: "Compute tier override; defaults to the service's plan."},
			"image":   schema.StringAttribute{Optional: true, Description: "Container image reference (e.g. nginx:latest) for image-based deployments."},
			"image_type": schema.StringAttribute{
				Optional:    true,
				Description: "Image visibility: \"public\" (default) or \"private\". Set to \"private\" to pull from an authenticated registry using image_provider/image_username/image_password/image_url.",
			},
			"image_provider": schema.StringAttribute{
				Optional:    true,
				Description: "Registry provider for a private image (e.g. azure, docker, other, custom). Determines how credentials are applied.",
			},
			"image_username": schema.StringAttribute{
				Optional:    true,
				Description: "Registry username for a private image.",
			},
			"image_password": schema.StringAttribute{
				Optional:    true,
				Sensitive:   true,
				Description: "Registry password or access token for a private image. Never logged.",
			},
			"image_url": schema.StringAttribute{
				Optional:    true,
				Description: "Registry endpoint for a private image. Required for providers that authenticate with username + password (azure, docker, other, custom).",
			},
			"build_args": schema.MapAttribute{
				Optional:    true,
				ElementType: types.StringType,
				Description: "PUBLIC docker build args passed to the build. Values are baked into image layers, so they must never carry secrets (use secret_vars for those). Merged over any auto-derived args; an explicit key wins on collision.",
			},
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
