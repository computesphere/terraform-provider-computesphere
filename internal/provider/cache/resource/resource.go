package provider

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"time"

	csv2 "github.com/computesphere/computesphere-go"
	"github.com/computesphere/terraform-provider-computesphere/internal/provider/common/dataproduct"
	cstypes "github.com/computesphere/terraform-provider-computesphere/internal/provider/types"
	"github.com/google/uuid"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/identityschema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// waitTimeout bounds how long create and resize wait for the cache to become
// available.
const waitTimeout = 30 * time.Minute

// pollInterval is how often the cache status is read while waiting.
var pollInterval = 10 * time.Second

// SetPollIntervalForTest shortens the status poll in tests. Test use only.
func SetPollIntervalForTest(d time.Duration) { pollInterval = d }

var (
	validEngines   = map[string]bool{"valkey": true}
	validVersions  = map[string]bool{"8.1": true}
	validEvictions = map[string]bool{"allkeys-lru": true, "noeviction": true}

	readyStatuses  = map[string]bool{"available": true}
	failedStatuses = map[string]bool{"failed": true, "stopped": true}
)

// apiCache is the v2 Cache object (GET/POST/PATCH /caches).
type apiCache struct {
	ID               string  `json:"id"`
	Name             string  `json:"name"`
	Engine           string  `json:"engine"`
	Version          string  `json:"version"`
	InstanceType     *string `json:"instance_type"`
	HighAvailability bool    `json:"high_availability"`
	Eviction         *string `json:"eviction"`
	Status           string  `json:"status"`
	ReadOnly         bool    `json:"read_only"`
	ProjectID        string  `json:"project_id"`
	EnvironmentID    string  `json:"environment_id"`
	Region           string  `json:"region"`
	Error            string  `json:"error"`
	CreatedAt        string  `json:"created_at"`
}

// createCacheRequest is the closed CreateCacheRequest body.
type createCacheRequest struct {
	Name          string  `json:"name"`
	EnvironmentID string  `json:"environment_id"`
	Engine        string  `json:"engine"`
	Version       *string `json:"version,omitempty"`
	InstanceType  string  `json:"instance_type"`
	Eviction      *string `json:"eviction,omitempty"`
}

// updateCacheRequest is the closed UpdateCacheRequest body.
type updateCacheRequest struct {
	InstanceType *string `json:"instance_type,omitempty"`
	Eviction     *string `json:"eviction,omitempty"`
}

type CacheResource struct {
	client    *csv2.ClientWithResponses
	accountID string
}

var _ resource.Resource = &CacheResource{}
var _ resource.ResourceWithImportState = &CacheResource{}
var _ resource.ResourceWithIdentity = &CacheResource{}
var _ resource.ResourceWithValidateConfig = &CacheResource{}
var _ resource.ResourceWithModifyPlan = &CacheResource{}

func NewCacheResource() resource.Resource {
	return &CacheResource{}
}

func (r *CacheResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = "computesphere_cache"
}

func (r *CacheResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = Schema(ctx)
}

type cacheResourceModel struct {
	ID               types.String `tfsdk:"id"`
	Name             types.String `tfsdk:"name"`
	EnvironmentID    types.String `tfsdk:"environment_id"`
	Engine           types.String `tfsdk:"engine"`
	Version          types.String `tfsdk:"version"`
	InstanceType     types.String `tfsdk:"instance_type"`
	Eviction         types.String `tfsdk:"eviction"`
	HighAvailability types.Bool   `tfsdk:"high_availability"`
	Status           types.String `tfsdk:"status"`
	Region           types.String `tfsdk:"region"`
	ProjectID        types.String `tfsdk:"project_id"`
	CreatedAt        types.String `tfsdk:"created_at"`
}

func (r *CacheResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	data := cstypes.ConfigureResource(req, resp)
	if data != nil {
		r.client = data.V2Client
		r.accountID = data.AccountID
	}
}

func known(v types.String) bool { return !v.IsNull() && !v.IsUnknown() && v.ValueString() != "" }

// ValidateConfig mirrors the closed create body so mistakes surface at plan
// time instead of as a 400 on apply.
func (r *CacheResource) ValidateConfig(ctx context.Context, req resource.ValidateConfigRequest, resp *resource.ValidateConfigResponse) {
	var c cacheResourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &c)...)
	if resp.Diagnostics.HasError() {
		return
	}
	if known(c.Name) && !dataproduct.ValidName(c.Name.ValueString()) {
		resp.Diagnostics.AddAttributeError(path.Root("name"), "Invalid name",
			"name must be 1-40 lowercase letters, digits and hyphens, start with a letter and not end with a hyphen.")
	}
	check := func(attr string, v types.String, ok map[string]bool, list string) {
		if known(v) && !ok[v.ValueString()] {
			resp.Diagnostics.AddAttributeError(path.Root(attr), "Invalid value", fmt.Sprintf("%s must be one of %s.", attr, list))
		}
	}
	check("engine", c.Engine, validEngines, "valkey")
	check("version", c.Version, validVersions, "8.1")
	check("eviction", c.Eviction, validEvictions, "allkeys-lru, noeviction")
}

// apply maps a v2 Cache into Terraform state.
func (m *cacheResourceModel) apply(c *apiCache) {
	m.ID = types.StringValue(c.ID)
	m.Name = types.StringValue(c.Name)
	m.EnvironmentID = types.StringValue(c.EnvironmentID)
	m.Engine = types.StringValue(c.Engine)
	m.Version = types.StringValue(c.Version)
	m.InstanceType = types.StringPointerValue(c.InstanceType)
	m.Eviction = types.StringPointerValue(c.Eviction)
	m.HighAvailability = types.BoolValue(c.HighAvailability)
	m.Status = types.StringValue(c.Status)
	m.Region = types.StringValue(c.Region)
	m.ProjectID = types.StringValue(c.ProjectID)
	m.CreatedAt = types.StringValue(c.CreatedAt)
}

func (r *CacheResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan cacheResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	if _, err := uuid.Parse(r.accountID); err != nil {
		resp.Diagnostics.AddError("Invalid account_id", err.Error())
		return
	}
	if _, err := uuid.Parse(plan.EnvironmentID.ValueString()); err != nil {
		resp.Diagnostics.AddAttributeError(path.Root("environment_id"), "Invalid environment_id", err.Error())
		return
	}

	body := createCacheRequest{
		Name:          plan.Name.ValueString(),
		EnvironmentID: plan.EnvironmentID.ValueString(),
		Engine:        plan.Engine.ValueString(),
		InstanceType:  plan.InstanceType.ValueString(),
	}
	if known(plan.Version) {
		body.Version = cstypes.StringPtr(plan.Version.ValueString())
	}
	if known(plan.Eviction) {
		body.Eviction = cstypes.StringPtr(plan.Eviction.ValueString())
	}

	var created apiCache
	status, raw, err := cstypes.DoJSON(ctx, r.client, http.MethodPost, "/caches", nil, body, &created)
	if err != nil {
		resp.Diagnostics.AddError("Error creating cache", err.Error())
		return
	}
	if status != http.StatusCreated {
		resp.Diagnostics.AddError("Error creating cache", cstypes.ProblemSummary(raw, status))
		return
	}

	state := plan
	state.apply(&created)
	// Save the id first so a failed wait doesn't orphan the cache.
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	if c := r.waitAvailable(ctx, created.ID, &resp.Diagnostics); c != nil {
		state.apply(c)
		resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
	}
}

// get reads a cache; gone is true on 404.
func (r *CacheResource) get(ctx context.Context, id string) (*apiCache, bool, error) {
	var c apiCache
	status, raw, err := cstypes.DoJSON(ctx, r.client, http.MethodGet, "/caches/"+url.PathEscape(id), nil, nil, &c)
	if err != nil {
		return nil, false, err
	}
	if status == http.StatusNotFound {
		return nil, true, nil
	}
	if status != http.StatusOK {
		return nil, false, fmt.Errorf("%s", cstypes.ProblemSummary(raw, status))
	}
	return &c, false, nil
}

// waitAvailable polls until the cache is available. It returns the cache,
// or nil with a diagnostic when it fails or the wait times out.
func (r *CacheResource) waitAvailable(ctx context.Context, id string, diags *diag.Diagnostics) *apiCache {
	var last *apiCache
	ok := dataproduct.Wait(ctx, "cache", waitTimeout, pollInterval, readyStatuses, failedStatuses,
		func(ctx context.Context) (dataproduct.Snapshot, error) {
			c, gone, err := r.get(ctx, id)
			if err != nil || gone {
				return dataproduct.Snapshot{Gone: gone}, err
			}
			last = c
			return dataproduct.Snapshot{Status: c.Status, Error: c.Error}, nil
		}, diags)
	if !ok {
		return nil
	}
	return last
}

func (r *CacheResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state cacheResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	c, gone, err := r.get(ctx, state.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Error reading cache", err.Error())
		return
	}
	if gone {
		resp.State.RemoveResource(ctx)
		return
	}
	if c.ReadOnly {
		resp.Diagnostics.AddError("Cache is managed from its service",
			"This Valkey cache was created as a service before SphereCache, so it can't be managed with "+
				"computesphere_cache. Manage it from its service instead.")
		return
	}
	state.apply(c)
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *CacheResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan, state cacheResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	if !dataproduct.RefuseChangeInUpdate(ctx, req, resp, "cache", "name", "environment_id", "engine", "version") {
		return
	}
	id := state.ID.ValueString()

	var body updateCacheRequest
	changed := false
	if known(plan.InstanceType) && plan.InstanceType.ValueString() != state.InstanceType.ValueString() {
		body.InstanceType = cstypes.StringPtr(plan.InstanceType.ValueString())
		changed = true
	}
	if known(plan.Eviction) && plan.Eviction.ValueString() != state.Eviction.ValueString() {
		body.Eviction = cstypes.StringPtr(plan.Eviction.ValueString())
		changed = true
	}

	newState := plan
	if changed {
		var updated apiCache
		status, raw, err := cstypes.DoJSON(ctx, r.client, http.MethodPatch, "/caches/"+url.PathEscape(id), nil, body, &updated)
		if err != nil {
			resp.Diagnostics.AddError("Error updating cache", err.Error())
			return
		}
		if status != http.StatusOK {
			resp.Diagnostics.AddError("Error updating cache", cstypes.ProblemSummary(raw, status))
			return
		}
		newState.apply(&updated)
		if c := r.waitAvailable(ctx, id, &resp.Diagnostics); c != nil {
			newState.apply(c)
		}
	} else if c, _, err := r.get(ctx, id); err == nil && c != nil {
		newState.apply(c)
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &newState)...)
}

func (r *CacheResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state cacheResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	status, raw, err := cstypes.DoJSON(ctx, r.client, http.MethodDelete, "/caches/"+url.PathEscape(state.ID.ValueString()), nil, nil, nil)
	if err != nil {
		resp.Diagnostics.AddError("Error deleting cache", err.Error())
		return
	}
	switch status {
	case http.StatusOK, http.StatusAccepted, http.StatusNoContent, http.StatusNotFound:
		resp.State.RemoveResource(ctx)
	default:
		resp.Diagnostics.AddError("Error deleting cache", cstypes.ProblemSummary(raw, status))
	}
}

// ModifyPlan refuses a name, environment, engine or version change on an
// existing cache (no update body takes them)
// instead of planning a replacement that would delete its data.
func (r *CacheResource) ModifyPlan(ctx context.Context, req resource.ModifyPlanRequest, resp *resource.ModifyPlanResponse) {
	dataproduct.RefuseChange(ctx, req, resp, "cache", "name", "environment_id", "engine", "version")
}

func (r *CacheResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

func (r *CacheResource) IdentitySchema(ctx context.Context, req resource.IdentitySchemaRequest, resp *resource.IdentitySchemaResponse) {
	resp.IdentitySchema = identityschema.Schema{
		Attributes: map[string]identityschema.Attribute{
			"id": identityschema.StringAttribute{RequiredForImport: true},
		},
	}
}
