package provider

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
)

// model holds every attribute of every compute kind. A kind's plan, state
// or config is read into it with fromObject (attributes the kind lacks stay
// null) and written back with toObject (only the kind's attributes are
// kept), so one implementation serves all five resources.
type model struct {
	ID            types.String `tfsdk:"id"`
	Name          types.String `tfsdk:"name"`
	Status        types.String `tfsdk:"status"`
	ServiceID     types.String `tfsdk:"service_id"`
	EnvironmentID types.String `tfsdk:"environment_id"`
	ProjectID     types.String `tfsdk:"project_id"`
	EnvVars       types.Map    `tfsdk:"env_vars"`
	SecretVars    types.Map    `tfsdk:"secret_vars"`
	WaitForReady  types.Bool   `tfsdk:"wait_for_ready"`
	DeployPolicy  types.String `tfsdk:"deploy_policy"`
	TriggerOn     types.String `tfsdk:"trigger_on"`
	TagPattern    types.String `tfsdk:"tag_pattern"`

	Port        types.Int64 `tfsdk:"port"`
	SphereCount types.Int64 `tfsdk:"sphere_count"`

	Image         types.String `tfsdk:"image"`
	ImageType     types.String `tfsdk:"image_type"`
	ImageProvider types.String `tfsdk:"image_provider"`
	ImageUsername types.String `tfsdk:"image_username"`
	ImagePassword types.String `tfsdk:"image_password"`
	ImagePwVer    types.Int64  `tfsdk:"image_password_wo_version"`
	ImageURL      types.String `tfsdk:"image_url"`

	GitURL           types.String `tfsdk:"git_url"`
	Branch           types.String `tfsdk:"branch"`
	GitConnectionID  types.String `tfsdk:"git_connection_id"`
	GitRepoID        types.String `tfsdk:"git_repo_id"`
	RootDir          types.String `tfsdk:"root_dir"`
	RootDirectory    types.String `tfsdk:"root_directory"`
	WatchPaths       types.List   `tfsdk:"watch_paths"`
	BuildArgs        types.Map    `tfsdk:"build_args"`
	RuntimeType      types.String `tfsdk:"runtime_type"`
	Language         types.String `tfsdk:"language"`
	Version          types.String `tfsdk:"version"`
	BuildCommand     types.String `tfsdk:"build_command"`
	StartCommand     types.String `tfsdk:"start_command"`
	Dockerfile       types.String `tfsdk:"dockerfile"`
	Context          types.String `tfsdk:"context"`
	InstallCommand   types.String `tfsdk:"install_command"`
	PublishDirectory types.String `tfsdk:"publish_directory"`
	Spa              types.Bool   `tfsdk:"spa"`

	Command          types.String `tfsdk:"command"`
	Schedule         types.String `tfsdk:"schedule"`
	PredeployCommand types.String `tfsdk:"predeploy_command"`

	HealthCheck   types.Object `tfsdk:"health_check"`
	Autoscaling   types.Object `tfsdk:"autoscaling"`
	NetworkAccess types.Object `tfsdk:"network_access"`

	URL         types.String `tfsdk:"url"`
	InternalURL types.String `tfsdk:"internal_url"`
}

type healthCheckModel struct {
	Enabled                   types.Bool   `tfsdk:"enabled"`
	Path                      types.String `tfsdk:"path"`
	Port                      types.Int64  `tfsdk:"port"`
	InitialDelaySeconds       types.Int64  `tfsdk:"initial_delay_seconds"`
	PeriodSeconds             types.Int64  `tfsdk:"period_seconds"`
	TimeoutSeconds            types.Int64  `tfsdk:"timeout_seconds"`
	ReadinessFailureThreshold types.Int64  `tfsdk:"readiness_failure_threshold"`
}

type autoscalingModel struct {
	Enabled        types.Bool  `tfsdk:"enabled"`
	MinReplicas    types.Int64 `tfsdk:"min_replicas"`
	MaxReplicas    types.Int64 `tfsdk:"max_replicas"`
	CPUUtilization types.Int64 `tfsdk:"cpu_utilization"`
}

type networkAccessModel struct {
	Type               types.String `tfsdk:"type"`
	WhitelistedIPs     types.List   `tfsdk:"whitelisted_ips"`
	DefaultURLDisabled types.Bool   `tfsdk:"default_url_disabled"`
}

// allAttrTypes is the object type of model.
func allAttrTypes() map[string]attr.Type {
	t, ok := allAttrsSchema().Type().(basetypes.ObjectType)
	if !ok {
		panic("compute schema type is not an object")
	}
	return t.AttrTypes
}

// fromObject reads a kind's object (plan, state or config) into a model.
// Attributes the kind doesn't have are null.
func fromObject(ctx context.Context, obj types.Object) (model, diag.Diagnostics) {
	var m model
	all := allAttrTypes()
	vals := make(map[string]attr.Value, len(all))
	given := obj.Attributes()
	for name, t := range all {
		if v, ok := given[name]; ok {
			vals[name] = v
			continue
		}
		vals[name] = nullOf(t)
	}
	full, diags := types.ObjectValue(all, vals)
	if diags.HasError() {
		return m, diags
	}
	diags.Append(full.As(ctx, &m, basetypes.ObjectAsOptions{})...)
	return m, diags
}

// toObject writes a model as the object of kind k, keeping only k's
// attributes.
func toObject(ctx context.Context, k kind, m model) (types.Object, diag.Diagnostics) {
	full, diags := types.ObjectValueFrom(ctx, allAttrTypes(), m)
	if diags.HasError() {
		return types.ObjectNull(nil), diags
	}
	kt, ok := Schema(k).Type().(basetypes.ObjectType)
	if !ok {
		diags.AddError("Internal error", "compute schema type is not an object")
		return types.ObjectNull(nil), diags
	}
	vals := make(map[string]attr.Value, len(kt.AttrTypes))
	for name := range kt.AttrTypes {
		vals[name] = full.Attributes()[name]
	}
	out, d := types.ObjectValue(kt.AttrTypes, vals)
	diags.Append(d...)
	return out, diags
}

func nullOf(t attr.Type) attr.Value {
	switch tt := t.(type) {
	case basetypes.StringType:
		return types.StringNull()
	case basetypes.Int64Type:
		return types.Int64Null()
	case basetypes.BoolType:
		return types.BoolNull()
	case basetypes.ListType:
		return types.ListNull(tt.ElemType)
	case basetypes.MapType:
		return types.MapNull(tt.ElemType)
	case basetypes.ObjectType:
		return types.ObjectNull(tt.AttrTypes)
	}
	panic("unsupported attribute type in compute schema")
}
