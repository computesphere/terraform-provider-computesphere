package provider

import (
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/boolplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/listplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/mapplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/objectplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// attrDef builds one attribute for a kind; replace is true when the kind's
// change body can't change it, so a new value creates a new resource.
type attrDef func(k kind, replace bool) schema.Attribute

func strMods(replace bool) []planmodifier.String {
	if replace {
		return []planmodifier.String{stringplanmodifier.RequiresReplace()}
	}
	return nil
}

func int64Mods(replace bool) []planmodifier.Int64 {
	if replace {
		return []planmodifier.Int64{int64planmodifier.RequiresReplace()}
	}
	return nil
}

func boolMods(replace bool) []planmodifier.Bool {
	if replace {
		return []planmodifier.Bool{boolplanmodifier.RequiresReplace()}
	}
	return nil
}

func listMods(replace bool) []planmodifier.List {
	if replace {
		return []planmodifier.List{listplanmodifier.RequiresReplace()}
	}
	return nil
}

func mapMods(replace bool) []planmodifier.Map {
	if replace {
		return []planmodifier.Map{mapplanmodifier.RequiresReplace()}
	}
	return nil
}

func objMods(replace bool) []planmodifier.Object {
	if replace {
		return []planmodifier.Object{objectplanmodifier.RequiresReplace()}
	}
	return nil
}

// describe appends the replacement note to a description when needed.
func describe(k kind, replace bool, text string) string {
	if replace {
		return text + fmt.Sprintf(" Changing it creates a new %s.", k.noun)
	}
	return text
}

func optString(text string) attrDef {
	return func(k kind, replace bool) schema.Attribute {
		return schema.StringAttribute{Optional: true, PlanModifiers: strMods(replace), Description: describe(k, replace, text)}
	}
}

func optInt64(text string) attrDef {
	return func(k kind, replace bool) schema.Attribute {
		return schema.Int64Attribute{Optional: true, PlanModifiers: int64Mods(replace), Description: describe(k, replace, text)}
	}
}

func optBool(text string) attrDef {
	return func(k kind, replace bool) schema.Attribute {
		return schema.BoolAttribute{Optional: true, PlanModifiers: boolMods(replace), Description: describe(k, replace, text)}
	}
}

func computedString(text string) attrDef {
	return func(k kind, _ bool) schema.Attribute {
		return schema.StringAttribute{Computed: true, Description: text,
			PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()}}
	}
}

var attrDefs = map[string]attrDef{
	"id": func(k kind, _ bool) schema.Attribute {
		return schema.StringAttribute{Computed: true, Description: "Deployment id of the " + k.noun + ".",
			PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()}}
	},
	"name": computedString("Name, taken from the service."),
	"status": func(k kind, _ bool) schema.Attribute {
		return schema.StringAttribute{Computed: true, Description: "Current status."}
	},
	"service_id": func(k kind, _ bool) schema.Attribute {
		return schema.StringAttribute{Required: true, PlanModifiers: strMods(true),
			Description: describe(k, true, "Service this "+k.noun+" runs; its name and plan are taken from the service.")}
	},
	"environment_id": func(k kind, _ bool) schema.Attribute {
		return schema.StringAttribute{Required: true, PlanModifiers: strMods(true),
			Description: describe(k, true, "Environment the "+k.noun+" runs in.")}
	},
	"project_id": func(k kind, _ bool) schema.Attribute {
		return schema.StringAttribute{Required: true, PlanModifiers: strMods(true),
			Description: describe(k, true, "Project the service belongs to.")}
	},
	"env_vars": func(k kind, _ bool) schema.Attribute {
		return schema.MapAttribute{Optional: true, ElementType: types.StringType,
			Description: "Plain environment variables, name to value. Changing them redeploys."}
	},
	"secret_vars": func(k kind, _ bool) schema.Attribute {
		return schema.MapAttribute{Optional: true, Sensitive: true, ElementType: types.StringType,
			Description: "Secret environment variables, name to value. Changing them redeploys."}
	},
	"wait_for_ready": func(k kind, _ bool) schema.Attribute {
		return schema.BoolAttribute{Optional: true, Computed: true, Default: booldefault.StaticBool(true),
			Description: "Wait for the " + k.noun + " to finish deploying after create and change."}
	},
	"deploy_policy": optString("Deploy policy: `auto` or `approval`. Leave it out to use the " +
		"environment's policy. Setting it needs the project admin role."),
	"trigger_on": optString("What a git push must be to build: `branch` (pushes to `branch`, the default) or " +
		"`tag` (tags matching `tag_pattern`)."),
	"tag_pattern": optString("Glob a tag must match to build when `trigger_on` is `tag`, e.g. `v*`."),

	"port":         optInt64("Port the service listens on. Defaults to 8080."),
	"sphere_count": optInt64("Spherelets to run. Defaults to 1. Changing it scales in place."),
	"image":        optString("Container image to deploy, e.g. `nginx:1.27`. Set this or `git_url`."),
	"image_type": optString("`public` (the default) or `private`. A private image is pulled with " +
		"`image_provider`, `image_username`, `image_password` and `image_url`."),
	"image_provider": optString("Registry provider of a private image, e.g. `docker`, `azure`, `other` or `custom`."),
	"image_username": optString("Registry username of a private image."),
	"image_password": func(k kind, _ bool) schema.Attribute {
		return schema.StringAttribute{Optional: true, Sensitive: true, WriteOnly: true,
			Description: "Registry password or token of a private image. Write-only: it is sent to the API and " +
				"never stored in state or plan (Terraform 1.11 or later). It goes with every image change, because " +
				"the API keeps a stored password only while the registry stays the same. To rotate it, set the " +
				"new value and increase `image_password_wo_version`."}
	},
	"image_password_wo_version": func(k kind, _ bool) schema.Attribute {
		return schema.Int64Attribute{Optional: true,
			Description: "Version of `image_password`. Increase it to send a new password; the change applies in " +
				"place. Setting it for the first time on an imported or moved resource only records it."}
	},
	"image_url": optString("Registry endpoint of a private image. Needed for providers that sign in with a " +
		"username and password."),

	"git_url":           optString("Repository to build from. Set this or `image`."),
	"branch":            optString("Branch to build and to deploy on push."),
	"git_connection_id": optString("Git connection that clones the repository and delivers pushes."),
	"git_repo_id": func(k kind, _ bool) schema.Attribute {
		return schema.StringAttribute{Optional: true, Computed: true,
			PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			Description: "The repository's id at its git provider, e.g. a GitLab project id. Needed for GitLab " +
				"sources. Sent at create, and read back. If the " + k.noun + " has none at ComputeSphere (e.g. one " +
				"created without it), setting it only records it here. Removing it from the configuration keeps " +
				"the recorded value, and a value that differs from the " + k.noun + "'s is refused at plan time, " +
				"because the API can't change it."}
	},
	"root_dir":       optString("The service's directory in a monorepo."),
	"root_directory": optString("The site's directory in a monorepo."),
	"watch_paths": func(k kind, replace bool) schema.Attribute {
		return schema.ListAttribute{Optional: true, ElementType: types.StringType, PlanModifiers: listMods(replace),
			Description: describe(k, replace, "gitignore-style globs; a push builds only when a changed file "+
				"matches one. Defaults to `root_dir`, else every push.")}
	},
	"build_args": func(k kind, replace bool) schema.Attribute {
		return schema.MapAttribute{Optional: true, ElementType: types.StringType, PlanModifiers: mapMods(replace),
			Description: describe(k, replace, "Docker build args. Public values only: they are baked into the "+
				"image, so secrets belong in `secret_vars`.")}
	},
	"runtime_type": optString("Build mode of a git source: `native` (a managed toolchain set by `language`) or " +
		"`docker` (a Dockerfile). Derived from the source when left out."),
	"language":      optString("Managed toolchain of a native build, e.g. `nodejs`, `python` or `go`."),
	"version":       optString("Language version of a native build, e.g. `20` for Node.js."),
	"build_command": optString("Build command."),
	"start_command": optString("Start command of a git build."),
	"dockerfile":    optString("Dockerfile path of a Docker build."),
	"context":       optString("Docker build context directory."),

	"install_command":   optString("Dependency install command, e.g. `npm ci`."),
	"publish_directory": optString("Directory the build emits and that gets published, e.g. `dist`. Defaults to `dist`."),
	"spa": optBool("Single-page-app fallback: serve `index.html` for paths that match no file. Defaults to " +
		"true. Changes apply to the live site without a build."),

	"command": func(k kind, replace bool) schema.Attribute {
		desc := "Command to run instead of the image's own."
		if k.is(worker) {
			desc = "Command an image worker runs instead of the image's own. A worker built from `git_url` " +
				"runs `start_command` instead, so setting both is refused."
		}
		return schema.StringAttribute{Optional: true, PlanModifiers: strMods(replace), Description: describe(k, replace, desc)}
	},
	"schedule":          optString("Cron expression, evaluated in UTC unless it starts with `CRON_TZ=<zone>`, e.g. `CRON_TZ=Europe/London 0 9 * * *`."),
	"predeploy_command": optString("Command run before each new version starts serving, e.g. database migrations."),

	"health_check": func(k kind, replace bool) schema.Attribute {
		return schema.SingleNestedAttribute{Optional: true, PlanModifiers: objMods(replace),
			Description: describe(k, replace, "HTTP health check. A spherelet that fails it is taken out of traffic."),
			Attributes: map[string]schema.Attribute{
				"enabled":                     schema.BoolAttribute{Required: true, Description: "Turn the check on or off."},
				"path":                        schema.StringAttribute{Optional: true, Description: "HTTP path probed. Required while `enabled` is true."},
				"port":                        schema.Int64Attribute{Optional: true, Description: "Port probed; leave it out to probe the service port."},
				"initial_delay_seconds":       schema.Int64Attribute{Optional: true, Description: "Seconds before the first probe (0-300). Defaults to 10."},
				"period_seconds":              schema.Int64Attribute{Optional: true, Description: "Seconds between probes (0-300). Defaults to 30."},
				"timeout_seconds":             schema.Int64Attribute{Optional: true, Description: "Probe timeout in seconds (0-60); 0 uses the platform default of 3."},
				"readiness_failure_threshold": schema.Int64Attribute{Optional: true, Description: "Failures in a row before a spherelet is taken out of traffic (0-20); 0 uses the platform default of 3."},
			},
		}
	},
	"autoscaling": func(k kind, replace bool) schema.Attribute {
		return schema.SingleNestedAttribute{Optional: true, PlanModifiers: objMods(replace),
			Description: describe(k, replace, "CPU autoscaling between a fewest and a most spherelets. Team plan and above."),
			Attributes: map[string]schema.Attribute{
				"enabled":         schema.BoolAttribute{Required: true, Description: "Turn autoscaling on or off."},
				"min_replicas":    schema.Int64Attribute{Optional: true, Description: "Fewest spherelets."},
				"max_replicas":    schema.Int64Attribute{Optional: true, Description: "Most spherelets."},
				"cpu_utilization": schema.Int64Attribute{Optional: true, Description: "CPU percentage to scale at."},
			},
		}
	},
	"network_access": func(k kind, replace bool) schema.Attribute {
		return schema.SingleNestedAttribute{Optional: true, PlanModifiers: objMods(replace),
			Description: describe(k, replace, "Who can reach the "+k.noun+"."),
			Attributes: map[string]schema.Attribute{
				"type": schema.StringAttribute{Required: true,
					Description: "`public_access`, or `restricted_access` to allow only `whitelisted_ips`."},
				"whitelisted_ips": schema.ListAttribute{Optional: true, ElementType: types.StringType,
					Description: "Addresses or CIDR ranges allowed in when `type` is `restricted_access`."},
				"default_url_disabled": schema.BoolAttribute{Optional: true,
					Description: "Turn off the default ComputeSphere address. Needs a verified custom domain that is serving."},
			},
		}
	},

	"url":          computedString("Public address."),
	"internal_url": computedString("Address other services in the same environment use."),
}

// Schema builds the schema of one compute kind: the common attributes plus
// the kind's own, each marked to force replacement when the kind's change
// body doesn't take it.
func Schema(k kind) schema.Schema {
	attrs := map[string]schema.Attribute{}
	for name, def := range attrDefs {
		if k.has(name) {
			attrs[name] = def(k, k.forcesReplace(name))
		}
	}
	return schema.Schema{Description: k.summary, Attributes: attrs}
}

// allAttrsSchema is the union of every kind's attributes; the superset model
// is converted through its type.
func allAttrsSchema() schema.Schema {
	attrs := map[string]schema.Attribute{}
	for name, def := range attrDefs {
		attrs[name] = def(webService, false)
	}
	return schema.Schema{Attributes: attrs}
}
