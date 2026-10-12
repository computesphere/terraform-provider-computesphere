package provider

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
)

// The bodies are plain maps so that a field is sent only when the kind has
// it and it is set, and a field removed from the configuration can be sent
// as JSON null (the change body's way to clear a saved build setting).
// Server-only fields (tag, is_build_complete, webhook_id, pipeline_tier, a
// plan override, custom_domain, db_config, ci_config) are never sent.

func setString(body map[string]any, key string, v types.String) {
	if !v.IsNull() && !v.IsUnknown() && v.ValueString() != "" {
		body[key] = v.ValueString()
	}
}

func setInt64(body map[string]any, key string, v types.Int64) {
	if !v.IsNull() && !v.IsUnknown() {
		body[key] = v.ValueInt64()
	}
}

func setBool(body map[string]any, key string, v types.Bool) {
	if !v.IsNull() && !v.IsUnknown() {
		body[key] = v.ValueBool()
	}
}

func setStringMap(ctx context.Context, body map[string]any, key string, v types.Map, diags *diag.Diagnostics) {
	if v.IsNull() || v.IsUnknown() {
		return
	}
	out := map[string]string{}
	diags.Append(v.ElementsAs(ctx, &out, false)...)
	body[key] = out
}

func setStringList(ctx context.Context, body map[string]any, key string, v types.List, diags *diag.Diagnostics) {
	if v.IsNull() || v.IsUnknown() {
		return
	}
	out := []string{}
	diags.Append(v.ElementsAs(ctx, &out, false)...)
	body[key] = out
}

func healthCheckBody(ctx context.Context, v types.Object, diags *diag.Diagnostics) map[string]any {
	var hc healthCheckModel
	diags.Append(v.As(ctx, &hc, basetypes.ObjectAsOptions{})...)
	b := map[string]any{}
	setBool(b, "enabled", hc.Enabled)
	setString(b, "path", hc.Path)
	setInt64(b, "port", hc.Port)
	setInt64(b, "initial_delay_seconds", hc.InitialDelaySeconds)
	setInt64(b, "period_seconds", hc.PeriodSeconds)
	setInt64(b, "timeout_seconds", hc.TimeoutSeconds)
	setInt64(b, "readiness_failure_threshold", hc.ReadinessFailureThreshold)
	return b
}

func autoscalingBody(ctx context.Context, v types.Object, diags *diag.Diagnostics) map[string]any {
	var as autoscalingModel
	diags.Append(v.As(ctx, &as, basetypes.ObjectAsOptions{})...)
	b := map[string]any{}
	setBool(b, "enabled", as.Enabled)
	setInt64(b, "min_replicas", as.MinReplicas)
	setInt64(b, "max_replicas", as.MaxReplicas)
	setInt64(b, "cpu_utilization", as.CPUUtilization)
	return b
}

// networkBody renders network_access; withDefaultURL adds the change-only
// default_url_disabled flag.
func networkBody(ctx context.Context, v types.Object, withDefaultURL bool, diags *diag.Diagnostics) map[string]any {
	var na networkAccessModel
	diags.Append(v.As(ctx, &na, basetypes.ObjectAsOptions{})...)
	b := map[string]any{}
	setString(b, "type", na.Type)
	setStringList(ctx, b, "whitelisted_ips", na.WhitelistedIPs, diags)
	if withDefaultURL {
		setBool(b, "default_url_disabled", na.DefaultURLDisabled)
	}
	return b
}

func present(v attr.Value) bool { return !v.IsNull() && !v.IsUnknown() }

// createBody renders the kind's closed create body (POST /deployments).
func createBody(ctx context.Context, k kind, m model, diags *diag.Diagnostics) map[string]any {
	b := map[string]any{"type": k.apiType}
	setString(b, "service_id", m.ServiceID)
	setString(b, "environment_id", m.EnvironmentID)
	setString(b, "project_id", m.ProjectID)
	setStringMap(ctx, b, "env_vars", m.EnvVars, diags)
	setStringMap(ctx, b, "secret_vars", m.SecretVars, diags)

	if k.has("port") {
		setInt64(b, "port", m.Port)
	}
	if k.has("sphere_count") {
		setInt64(b, "sphere_count", m.SphereCount)
	}
	if k.has("image") {
		setString(b, "image", m.Image)
		setString(b, "image_type", m.ImageType)
		setString(b, "image_provider", m.ImageProvider)
		setString(b, "image_username", m.ImageUsername)
		setString(b, "image_password", m.ImagePassword)
		setString(b, "image_url", m.ImageURL)
	}
	if k.has("git_url") {
		setString(b, "git_url", m.GitURL)
		setString(b, "branch", m.Branch)
		setString(b, "git_connection_id", m.GitConnectionID)
		setString(b, "build_command", m.BuildCommand)
	}
	switch {
	case k.is(staticSite):
		b["build_source"] = "git"
	case present(m.GitURL) && m.GitURL.ValueString() != "":
		b["build_source"] = "git"
	case present(m.Image) && m.Image.ValueString() != "":
		b["build_source"] = "image"
	}
	if k.has("root_dir") {
		setString(b, "root_dir", m.RootDir)
		setStringList(ctx, b, "watch_paths", m.WatchPaths, diags)
		setStringMap(ctx, b, "build_args", m.BuildArgs, diags)
		setString(b, "runtime_type", m.RuntimeType)
		setString(b, "language", m.Language)
		setString(b, "version", m.Version)
		setString(b, "dockerfile", m.Dockerfile)
		setString(b, "context", m.Context)
	}
	if k.has("start_command") {
		setString(b, "start_command", m.StartCommand)
	}
	if k.has("root_directory") {
		setString(b, "root_directory", m.RootDirectory)
		setString(b, "install_command", m.InstallCommand)
		setString(b, "publish_directory", m.PublishDirectory)
		setBool(b, "spa", m.Spa)
	}
	if k.has("command") {
		setString(b, "command", m.Command)
	}
	if k.has("schedule") {
		setString(b, "schedule", m.Schedule)
	}
	if k.has("predeploy_command") {
		setString(b, "predeploy_command", m.PredeployCommand)
	}
	if k.has("health_check") && present(m.HealthCheck) {
		b["health_check"] = healthCheckBody(ctx, m.HealthCheck, diags)
	}
	if k.has("autoscaling") && present(m.Autoscaling) {
		b["auto_scaling_config"] = autoscalingBody(ctx, m.Autoscaling, diags)
	}
	// A static site takes network access only on change; it's applied
	// right after create.
	if k.is(webService) && present(m.NetworkAccess) {
		b["network_access"] = networkBody(ctx, m.NetworkAccess, false, diags)
	}
	return b
}

// postCreateBody renders the settings only the change body takes, applied
// right after create: the deploy settings, and network access on a static
// site (and the default-address switch on a web service).
func postCreateBody(ctx context.Context, k kind, plan model, diags *diag.Diagnostics) map[string]any {
	b := map[string]any{}
	setString(b, "deploy_policy", plan.DeployPolicy)
	setString(b, "trigger_on", plan.TriggerOn)
	setString(b, "tag_pattern", plan.TagPattern)
	if present(plan.NetworkAccess) {
		var na networkAccessModel
		diags.Append(plan.NetworkAccess.As(ctx, &na, basetypes.ObjectAsOptions{})...)
		switch {
		case k.is(staticSite):
			b["networking"] = networkBody(ctx, plan.NetworkAccess, true, diags)
		case k.is(webService) && present(na.DefaultURLDisabled):
			b["networking"] = map[string]any{"default_url_disabled": na.DefaultURLDisabled.ValueBool()}
		}
	}
	return b
}

// stringChange reports a changed string field: set to the new value, or to
// nil (JSON null) when it was removed from the configuration.
func stringChange(body map[string]any, key string, plan, state types.String) {
	if plan.IsUnknown() || plan.Equal(state) {
		return
	}
	if plan.IsNull() || plan.ValueString() == "" {
		if !state.IsNull() && state.ValueString() != "" {
			body[key] = nil
		}
		return
	}
	body[key] = plan.ValueString()
}

// changeBody renders the kind's closed change body (PATCH
// /deployments/{id}) from what differs between plan and state. Fields only
// the create takes are not here: they force replacement instead. A
// change-only setting given at create is sent with a nil state.
func changeBody(ctx context.Context, k kind, plan, state model, diags *diag.Diagnostics) map[string]any {
	b := map[string]any{}

	stringChange(b, "deploy_policy", plan.DeployPolicy, state.DeployPolicy)
	stringChange(b, "trigger_on", plan.TriggerOn, state.TriggerOn)
	stringChange(b, "tag_pattern", plan.TagPattern, state.TagPattern)

	if k.is(webService) || k.is(privateService) {
		if present(plan.Port) && !plan.Port.Equal(state.Port) {
			b["port"] = plan.Port.ValueInt64()
		}
		if imageChanged(plan, state) {
			img := map[string]any{}
			setString(img, "name", plan.Image)
			setString(img, "type", plan.ImageType)
			setString(img, "provider", plan.ImageProvider)
			setString(img, "username", plan.ImageUsername)
			setString(img, "password", plan.ImagePassword)
			setString(img, "url", plan.ImageURL)
			if len(img) > 0 {
				b["image"] = img
			}
		}
		if present(plan.HealthCheck) && !plan.HealthCheck.Equal(state.HealthCheck) {
			b["health_check"] = healthCheckBody(ctx, plan.HealthCheck, diags)
		} else if plan.HealthCheck.IsNull() && present(state.HealthCheck) {
			b["health_check"] = map[string]any{"enabled": false}
		}
		if present(plan.Autoscaling) && !plan.Autoscaling.Equal(state.Autoscaling) {
			b["autoscaling"] = autoscalingBody(ctx, plan.Autoscaling, diags)
		} else if plan.Autoscaling.IsNull() && present(state.Autoscaling) {
			b["autoscaling"] = map[string]any{"enabled": false}
		}
	}
	if (k.is(webService) || k.is(staticSite)) && present(plan.NetworkAccess) && !plan.NetworkAccess.Equal(state.NetworkAccess) {
		b["networking"] = networkBody(ctx, plan.NetworkAccess, true, diags)
	}
	if k.is(cronJob) {
		sched := map[string]any{}
		stringChange(sched, "cron", plan.Schedule, state.Schedule)
		stringChange(sched, "command", plan.Command, state.Command)
		if len(sched) > 0 {
			b["schedule"] = sched
		}
	}
	if !k.is(worker) {
		if bc := buildConfigChange(ctx, k, plan, state, diags); len(bc) > 0 {
			b["build_config"] = bc
		}
	}
	return b
}

func imageChanged(plan, state model) bool {
	return !plan.Image.Equal(state.Image) || !plan.ImageType.Equal(state.ImageType) ||
		!plan.ImageProvider.Equal(state.ImageProvider) || !plan.ImageUsername.Equal(state.ImageUsername) ||
		!plan.ImagePassword.Equal(state.ImagePassword) || !plan.ImageURL.Equal(state.ImageURL)
}

// buildConfigChange renders build_config: a merge, so only changed fields
// are sent and a removed field is sent as null.
func buildConfigChange(ctx context.Context, k kind, plan, state model, diags *diag.Diagnostics) map[string]any {
	bc := map[string]any{}
	stringChange(bc, "git_url", plan.GitURL, state.GitURL)
	stringChange(bc, "branch", plan.Branch, state.Branch)
	stringChange(bc, "git_connection_id", plan.GitConnectionID, state.GitConnectionID)
	stringChange(bc, "build_command", plan.BuildCommand, state.BuildCommand)
	if k.is(staticSite) {
		stringChange(bc, "root_directory", plan.RootDirectory, state.RootDirectory)
		stringChange(bc, "install_command", plan.InstallCommand, state.InstallCommand)
		stringChange(bc, "publish_directory", plan.PublishDirectory, state.PublishDirectory)
		if !plan.Spa.IsUnknown() && !plan.Spa.Equal(state.Spa) {
			if plan.Spa.IsNull() {
				bc["spa"] = nil
			} else {
				bc["spa"] = plan.Spa.ValueBool()
			}
		}
		return bc
	}
	stringChange(bc, "root_dir", plan.RootDir, state.RootDir)
	stringChange(bc, "build_type", plan.RuntimeType, state.RuntimeType)
	stringChange(bc, "language", plan.Language, state.Language)
	stringChange(bc, "version", plan.Version, state.Version)
	stringChange(bc, "dockerfile", plan.Dockerfile, state.Dockerfile)
	stringChange(bc, "context", plan.Context, state.Context)
	if k.has("start_command") {
		stringChange(bc, "run_command", plan.StartCommand, state.StartCommand)
	}
	if !plan.WatchPaths.IsUnknown() && !plan.WatchPaths.Equal(state.WatchPaths) {
		if plan.WatchPaths.IsNull() {
			bc["watch_paths"] = nil
		} else {
			setStringList(ctx, bc, "watch_paths", plan.WatchPaths, diags)
		}
	}
	if !plan.BuildArgs.IsUnknown() && !plan.BuildArgs.Equal(state.BuildArgs) {
		if plan.BuildArgs.IsNull() {
			bc["build_args"] = nil
		} else {
			setStringMap(ctx, bc, "build_args", plan.BuildArgs, diags)
		}
	}
	return bc
}
