// Package provider implements computesphere_secret_source: an account-level
// connection to an external secret manager (Doppler, Azure Key Vault) that
// deployments and environments read secrets from, so their values never
// pass through Terraform.
package provider

import (
	"context"
	"net/http"
	"net/url"

	csv2 "github.com/computesphere/computesphere-go"
	cstypes "github.com/computesphere/terraform-provider-computesphere/internal/provider/types"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/boolplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/setplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

type SecretSourceResource struct {
	client *csv2.ClientWithResponses
}

var (
	_ resource.Resource                = &SecretSourceResource{}
	_ resource.ResourceWithImportState = &SecretSourceResource{}
)

func NewSecretSourceResource() resource.Resource {
	return &SecretSourceResource{}
}

type model struct {
	ID                    types.String `tfsdk:"id"`
	Name                  types.String `tfsdk:"name"`
	Type                  types.String `tfsdk:"type"`
	Config                types.Map    `tfsdk:"config"`
	AuthType              types.String `tfsdk:"auth_type"`
	Credential            types.Map    `tfsdk:"credential"`
	CredentialVersion     types.Int64  `tfsdk:"credential_version"`
	AllProjects           types.Bool   `tfsdk:"all_projects"`
	ProjectIDs            types.Set    `tfsdk:"project_ids"`
	SyncIntervalS         types.Int64  `tfsdk:"sync_interval_s"`
	Status                types.String `tfsdk:"status"`
	CredentialFingerprint types.String `tfsdk:"credential_fingerprint"`
	RefCount              types.Int64  `tfsdk:"ref_count"`
}

// apiSource is the secret_source object, decoded from the raw response: the
// SDK in use has no secret-source operations.
type apiSource struct {
	ID                    string            `json:"id"`
	Name                  string            `json:"name"`
	Type                  string            `json:"type"`
	Status                string            `json:"status"`
	Config                map[string]string `json:"config"`
	AuthType              string            `json:"auth_type"`
	CredentialFingerprint *string           `json:"credential_fingerprint"`
	Access                *struct {
		AllProjects bool `json:"all_projects"`
		Projects    []struct {
			ProjectID string `json:"project_id"`
		} `json:"projects"`
	} `json:"access"`
	SyncIntervalS *int64 `json:"sync_interval_s"`
	RefCount      *int64 `json:"ref_count"`
}

func (r *SecretSourceResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = "computesphere_secret_source"
}

func (r *SecretSourceResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	keepStr := []planmodifier.String{stringplanmodifier.UseStateForUnknown()}
	resp.Schema = schema.Schema{
		Description: "Manages a secret manager connection for the account (Doppler or Azure Key Vault), which " +
			"deployments and environments read secrets from, so their values never pass through Terraform. Owners " +
			"and admins only. The connection is tested before it's saved: nothing is saved unless the secret manager " +
			"accepts the credential. A secret manager that secrets still read from can't be removed.",
		Attributes: map[string]schema.Attribute{
			"id":   schema.StringAttribute{Computed: true, PlanModifiers: keepStr, Description: "Secret manager id."},
			"name": schema.StringAttribute{Required: true, Description: "Name, unique in the account. Changes in place."},
			"type": schema.StringAttribute{
				Required: true, PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
				Description: "`doppler` or `azure-kv`. Changing it creates a new secret manager.",
			},
			"config": schema.MapAttribute{
				Required: true, ElementType: types.StringType,
				Description: "Settings of the type: Doppler `project` and `config`; Azure Key Vault `tenant_id` and " +
					"`vault`. Changes in place, after a connection test.",
			},
			"auth_type": schema.StringAttribute{
				Required:    true,
				Description: "How it signs in: `service_token` (Doppler) or `client_secret` (Azure Key Vault).",
			},
			"credential": schema.MapAttribute{
				Required: true, WriteOnly: true, Sensitive: true, ElementType: types.StringType,
				Description: "Sign-in fields of `auth_type`: Doppler `token`; Azure Key Vault `client_id` and " +
					"`client_secret`. Write-only: it's never stored in Terraform state, and the API never returns it. " +
					"It's sent on create, and again when `credential_version` or `auth_type` changes. Needs Terraform 1.11 or later.",
			},
			"credential_version": schema.Int64Attribute{
				Optional:    true,
				Description: "Change it to send `credential` again, for example after rotating the token.",
			},
			"all_projects": schema.BoolAttribute{
				Optional: true, Computed: true,
				PlanModifiers: []planmodifier.Bool{boolplanmodifier.UseStateForUnknown()},
				Description: "Let every project of the account use it, including projects an owner or admin creates later. " +
					"Can't be set with `project_ids`.",
			},
			"project_ids": schema.SetAttribute{
				Optional: true, Computed: true, ElementType: types.StringType,
				PlanModifiers: []planmodifier.Set{setplanmodifier.UseStateForUnknown()},
				Description: "Projects that may use it, each with all its environments. None by default; empty while " +
					"`all_projects` is true.",
			},
			"sync_interval_s": schema.Int64Attribute{
				Optional: true, Computed: true,
				PlanModifiers: []planmodifier.Int64{int64planmodifier.UseStateForUnknown()},
				Description:   "Seconds between checks for changed secret values.",
			},
			"status":                 schema.StringAttribute{Computed: true, Description: "`untested`, `ok` or `error`, from the last connection test."},
			"credential_fingerprint": schema.StringAttribute{Computed: true, Description: "A fingerprint of the stored credential, to tell whether it changed."},
			"ref_count":              schema.Int64Attribute{Computed: true, Description: "How many secrets read from it, in every project."},
		},
	}
}

func (r *SecretSourceResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if data := cstypes.ConfigureResource(req, resp); data != nil {
		r.client = data.V2Client
	}
}

func strMap(ctx context.Context, m types.Map, diags *diag.Diagnostics) map[string]string {
	out := map[string]string{}
	if m.IsNull() || m.IsUnknown() {
		return out
	}
	diags.Append(m.ElementsAs(ctx, &out, false)...)
	return out
}

// accessBody is the SecretSourceAccessRequest for the plan.
func accessBody(ctx context.Context, plan model, diags *diag.Diagnostics) map[string]any {
	all := !plan.AllProjects.IsNull() && !plan.AllProjects.IsUnknown() && plan.AllProjects.ValueBool()
	projects := []map[string]any{}
	if !all && !plan.ProjectIDs.IsNull() && !plan.ProjectIDs.IsUnknown() {
		var ids []string
		diags.Append(plan.ProjectIDs.ElementsAs(ctx, &ids, false)...)
		for _, id := range ids {
			projects = append(projects, map[string]any{"project_id": id})
		}
	}
	return map[string]any{"all_projects": all, "projects": projects}
}

func (m *model) apply(ctx context.Context, s *apiSource, diags *diag.Diagnostics) {
	m.ID = types.StringValue(s.ID)
	m.Name = types.StringValue(s.Name)
	m.Type = types.StringValue(s.Type)
	if s.Config != nil {
		v, d := types.MapValueFrom(ctx, types.StringType, s.Config)
		diags.Append(d...)
		m.Config = v
	}
	if s.AuthType != "" {
		m.AuthType = types.StringValue(s.AuthType)
	}
	m.Status = types.StringValue(s.Status)
	m.CredentialFingerprint = types.StringPointerValue(s.CredentialFingerprint)
	m.RefCount = types.Int64PointerValue(s.RefCount)
	m.SyncIntervalS = types.Int64PointerValue(s.SyncIntervalS)
	ids := []string{}
	all := false
	if s.Access != nil {
		all = s.Access.AllProjects
		// With all_projects the API lists every project; project_ids is
		// then empty, as configured.
		if !all {
			for _, p := range s.Access.Projects {
				ids = append(ids, p.ProjectID)
			}
		}
	}
	m.AllProjects = types.BoolValue(all)
	v, d := types.SetValueFrom(ctx, types.StringType, ids)
	diags.Append(d...)
	m.ProjectIDs = v
	// The credential is write-only: never kept in state.
	m.Credential = types.MapNull(types.StringType)
}

func (r *SecretSourceResource) do(ctx context.Context, method, p string, in any, ok int, what string, diags *diag.Diagnostics) *apiSource {
	var out apiSource
	status, raw, err := cstypes.DoJSON(ctx, r.client, method, p, nil, in, &out)
	if err != nil {
		diags.AddError("Error "+what+" secret manager", err.Error())
		return nil
	}
	if status != ok {
		diags.AddError("Error "+what+" secret manager", cstypes.ProblemSummary(raw, status))
		return nil
	}
	return &out
}

func (r *SecretSourceResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan, cfg model
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(req.Config.Get(ctx, &cfg)...)
	if resp.Diagnostics.HasError() {
		return
	}
	body := map[string]any{
		"name": plan.Name.ValueString(), "type": plan.Type.ValueString(), "auth_type": plan.AuthType.ValueString(),
		"config": strMap(ctx, plan.Config, &resp.Diagnostics), "credential": strMap(ctx, cfg.Credential, &resp.Diagnostics),
		"access": accessBody(ctx, plan, &resp.Diagnostics),
	}
	if !plan.SyncIntervalS.IsNull() && !plan.SyncIntervalS.IsUnknown() {
		body["sync_interval_s"] = plan.SyncIntervalS.ValueInt64()
	}
	if resp.Diagnostics.HasError() {
		return
	}
	s := r.do(ctx, http.MethodPost, "/secret-sources", body, http.StatusCreated, "connecting", &resp.Diagnostics)
	if s == nil {
		return
	}
	state := plan
	state.apply(ctx, s, &resp.Diagnostics)
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *SecretSourceResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state model
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	var out apiSource
	status, raw, err := cstypes.DoJSON(ctx, r.client, http.MethodGet, "/secret-sources/"+url.PathEscape(state.ID.ValueString()), nil, nil, &out)
	if err != nil {
		resp.Diagnostics.AddError("Error reading secret manager", err.Error())
		return
	}
	if status == http.StatusNotFound {
		resp.State.RemoveResource(ctx)
		return
	}
	if status != http.StatusOK {
		resp.Diagnostics.AddError("Error reading secret manager", cstypes.ProblemSummary(raw, status))
		return
	}
	state.apply(ctx, &out, &resp.Diagnostics)
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *SecretSourceResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan, state, cfg model
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	resp.Diagnostics.Append(req.Config.Get(ctx, &cfg)...)
	if resp.Diagnostics.HasError() {
		return
	}
	id := url.PathEscape(state.ID.ValueString())
	body := map[string]any{}
	if !plan.Name.Equal(state.Name) {
		body["name"] = plan.Name.ValueString()
	}
	if !plan.Config.Equal(state.Config) {
		body["config"] = strMap(ctx, plan.Config, &resp.Diagnostics)
	}
	if !plan.AuthType.Equal(state.AuthType) || !plan.CredentialVersion.Equal(state.CredentialVersion) {
		body["auth_type"] = plan.AuthType.ValueString()
		body["credential"] = strMap(ctx, cfg.Credential, &resp.Diagnostics)
	}
	if !plan.SyncIntervalS.IsUnknown() && !plan.SyncIntervalS.Equal(state.SyncIntervalS) {
		body["sync_interval_s"] = plan.SyncIntervalS.ValueInt64()
	}
	if resp.Diagnostics.HasError() {
		return
	}
	if len(body) > 0 {
		if r.do(ctx, http.MethodPatch, "/secret-sources/"+id, body, http.StatusOK, "updating", &resp.Diagnostics) == nil {
			return
		}
	}
	if !plan.AllProjects.Equal(state.AllProjects) || !plan.ProjectIDs.Equal(state.ProjectIDs) {
		status, raw, err := cstypes.DoJSON(ctx, r.client, http.MethodPut, "/secret-sources/"+id+"/access", nil, accessBody(ctx, plan, &resp.Diagnostics), nil)
		if err != nil {
			resp.Diagnostics.AddError("Error changing secret manager access", err.Error())
			return
		}
		if status != http.StatusOK {
			resp.Diagnostics.AddError("Error changing secret manager access", cstypes.ProblemSummary(raw, status))
			return
		}
	}
	s := r.do(ctx, http.MethodGet, "/secret-sources/"+id, nil, http.StatusOK, "reading", &resp.Diagnostics)
	if s == nil {
		return
	}
	next := plan
	next.apply(ctx, s, &resp.Diagnostics)
	resp.Diagnostics.Append(resp.State.Set(ctx, &next)...)
}

func (r *SecretSourceResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state model
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	status, raw, err := cstypes.DoJSON(ctx, r.client, http.MethodDelete, "/secret-sources/"+url.PathEscape(state.ID.ValueString()), nil, nil, nil)
	if err != nil {
		resp.Diagnostics.AddError("Error removing secret manager", err.Error())
		return
	}
	switch status {
	case http.StatusOK, http.StatusNoContent, http.StatusNotFound:
	default:
		resp.Diagnostics.AddError("Error removing secret manager", cstypes.ProblemSummary(raw, status))
	}
}

// ImportState imports by id. The credential isn't returned: with
// credential_version set in the configuration, the first apply after an
// import sends the credential once.
func (r *SecretSourceResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

var _ resource.ResourceWithValidateConfig = &SecretSourceResource{}

// ValidateConfig refuses project_ids together with all_projects = true.
func (r *SecretSourceResource) ValidateConfig(ctx context.Context, req resource.ValidateConfigRequest, resp *resource.ValidateConfigResponse) {
	var c model
	resp.Diagnostics.Append(req.Config.Get(ctx, &c)...)
	if resp.Diagnostics.HasError() || c.AllProjects.IsUnknown() || !c.AllProjects.ValueBool() || c.ProjectIDs.IsNull() || c.ProjectIDs.IsUnknown() {
		return
	}
	if len(c.ProjectIDs.Elements()) > 0 {
		resp.Diagnostics.AddAttributeError(path.Root("project_ids"), "Conflicting attributes",
			"Set all_projects = true or list project_ids, not both.")
	}
}
