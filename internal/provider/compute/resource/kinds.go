// Package provider implements one Terraform resource per compute type
// (web service, private service, background worker, cron job, static site).
// Each resource exposes only the fields of its type's closed create and
// change bodies; fields the API can set only at create force a new resource.
package provider

// kind describes one compute type: its API type, its resource name, the
// attributes its schema has and which of them the API can't change after
// create (those force replacement).
type kind struct {
	apiType  string   // the `type` discriminator, e.g. "web-service"
	typeName string   // Terraform resource type name
	aliases  []string // older stored type values that mean the same type
	noun     string   // used in descriptions and diagnostics
	summary  string   // resource description
	attrs    []string // configurable and computed attributes beyond the common ones
	replace  []string // attributes the change body doesn't take
}

func (k kind) has(attr string) bool {
	if commonAttrs[attr] {
		return true
	}
	for _, a := range k.attrs {
		if a == attr {
			return true
		}
	}
	return false
}

func (k kind) forcesReplace(attr string) bool {
	for _, a := range k.replace {
		if a == attr {
			return true
		}
	}
	return false
}

// rotatesImagePassword reports whether the kind's change body can carry a
// new registry password. A worker's change body takes no image settings and
// a cron job's has no image object, so neither can today.
func (k kind) rotatesImagePassword() bool { return k.is(webService) || k.is(privateService) }

// is reports whether k and o are the same kind.
func (k kind) is(o kind) bool { return k.apiType == o.apiType }

// matches reports whether a deployment's stored type is this kind.
func (k kind) matches(apiType string) bool {
	if apiType == k.apiType {
		return true
	}
	for _, a := range k.aliases {
		if a == apiType {
			return true
		}
	}
	return false
}

// commonAttrs are on every compute resource.
var commonAttrs = map[string]bool{
	"id": true, "name": true, "status": true,
	"service_id": true, "environment_id": true, "project_id": true,
	"env_vars": true, "secret_vars": true, "wait_for_ready": true,
	"deploy_policy": true, "trigger_on": true, "tag_pattern": true,
}

var (
	imageAttrs = []string{"image", "image_type", "image_provider", "image_username", "image_password",
		"image_password_wo_version", "image_url"}
	// imageCredAttrs are the write-only registry password and its version:
	// rotating it is never a reason to replace a resource.
	imageCredAttrs = []string{"image_password", "image_password_wo_version"}
	gitAttrs       = []string{"git_url", "branch", "git_connection_id", "git_repo_id", "root_dir", "watch_paths", "build_args",
		"runtime_type", "language", "version", "build_command", "start_command", "dockerfile", "context"}
)

func join(groups ...[]string) []string {
	var out []string
	for _, g := range groups {
		out = append(out, g...)
	}
	return out
}

func without(in []string, drop ...string) []string {
	var out []string
outer:
	for _, a := range in {
		for _, d := range drop {
			if a == d {
				continue outer
			}
		}
		out = append(out, a)
	}
	return out
}

var (
	webService = kind{
		apiType: "web-service", typeName: "computesphere_web_service",
		aliases: []string{"deploy_app", "build_app"},
		noun:    "web service",
		summary: "Manages a web service in an environment: a service with a public address, deployed from a " +
			"container image or built from a git repository.",
		attrs: join(imageAttrs, gitAttrs, []string{"port", "sphere_count", "predeploy_command",
			"health_check", "autoscaling", "network_access", "url", "internal_url"}),
		replace: []string{"predeploy_command", "git_repo_id"},
	}
	privateService = kind{
		apiType: "private-service", typeName: "computesphere_private_service",
		noun: "private service",
		summary: "Manages a private service in an environment: a web service with no public address, reachable " +
			"only by other services in the environment.",
		attrs: join(imageAttrs, gitAttrs, []string{"port", "sphere_count", "predeploy_command",
			"health_check", "autoscaling", "internal_url"}),
		replace: []string{"predeploy_command", "git_repo_id"},
	}
	worker = kind{
		apiType: "background-worker", typeName: "computesphere_worker",
		noun: "background worker",
		summary: "Manages a background worker in an environment: a long-running process with no port, health " +
			"check or address, run from a container image or built from a git repository.",
		attrs: join(imageAttrs, gitAttrs, []string{"command", "sphere_count", "autoscaling"}),
		// A worker's change body takes only the deploy settings today, so its
		// image, command, build and autoscaling are set at create.
		replace: join(without(imageAttrs, imageCredAttrs...), gitAttrs, []string{"command", "autoscaling"}),
	}
	cronJob = kind{
		apiType: "cron-job", typeName: "computesphere_cron_job",
		aliases: []string{"cronjob"},
		noun:    "cron job",
		summary: "Manages a cron job in an environment: a command run on a schedule, from a container image or " +
			"a git build.",
		attrs:   join(imageAttrs, without(gitAttrs, "start_command"), []string{"schedule", "command"}),
		replace: join(without(imageAttrs, imageCredAttrs...), []string{"git_repo_id"}),
	}
	staticSite = kind{
		apiType: "static-site", typeName: "computesphere_static_site",
		noun: "static site",
		summary: "Manages a static site in an environment, always built from a git repository and served from " +
			"the directory the build emits.",
		attrs: []string{"git_url", "branch", "git_connection_id", "root_directory", "install_command",
			"build_command", "publish_directory", "spa", "network_access", "url"},
	}
)

// kinds lists every compute kind, in resource registration order.
var kinds = []kind{webService, privateService, worker, cronJob, staticSite}
