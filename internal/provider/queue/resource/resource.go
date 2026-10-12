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

// waitTimeout bounds how long create and resize wait for the broker to become
// available.
const waitTimeout = 30 * time.Minute

// pollInterval is how often the broker status is read while waiting.
var pollInterval = 10 * time.Second

// SetPollIntervalForTest shortens the status poll in tests. Test use only.
func SetPollIntervalForTest(d time.Duration) { pollInterval = d }

var (
	validEngines  = map[string]bool{"rabbitmq": true}
	validVersions = map[string]bool{"4.3": true}

	readyStatuses  = map[string]bool{"available": true}
	failedStatuses = map[string]bool{"failed": true}
)

// apiQueue is the v2 Queue object (GET/POST/PATCH /queues).
type apiQueue struct {
	ID            string  `json:"id"`
	Name          string  `json:"name"`
	Engine        string  `json:"engine"`
	Version       *string `json:"version"`
	InstanceType  *string `json:"instance_type"`
	Nodes         int64   `json:"nodes"`
	Status        string  `json:"status"`
	ReadOnly      bool    `json:"read_only"`
	ProjectID     string  `json:"project_id"`
	EnvironmentID string  `json:"environment_id"`
	Region        string  `json:"region"`
	Error         string  `json:"error"`
	CreatedAt     string  `json:"created_at"`
}

// createQueueRequest is the closed CreateQueueRequest body.
type createQueueRequest struct {
	Name          string  `json:"name"`
	EnvironmentID string  `json:"environment_id"`
	Engine        string  `json:"engine"`
	Version       *string `json:"version,omitempty"`
	InstanceType  string  `json:"instance_type"`
}

// updateQueueRequest is the closed UpdateQueueRequest body.
type updateQueueRequest struct {
	InstanceType *string `json:"instance_type,omitempty"`
}

type QueueResource struct {
	client    *csv2.ClientWithResponses
	accountID string
}

var _ resource.Resource = &QueueResource{}
var _ resource.ResourceWithImportState = &QueueResource{}
var _ resource.ResourceWithIdentity = &QueueResource{}
var _ resource.ResourceWithValidateConfig = &QueueResource{}
var _ resource.ResourceWithModifyPlan = &QueueResource{}

func NewQueueResource() resource.Resource {
	return &QueueResource{}
}

func (r *QueueResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = "computesphere_queue"
}

func (r *QueueResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = Schema(ctx)
}

type queueResourceModel struct {
	ID            types.String `tfsdk:"id"`
	Name          types.String `tfsdk:"name"`
	EnvironmentID types.String `tfsdk:"environment_id"`
	Engine        types.String `tfsdk:"engine"`
	Version       types.String `tfsdk:"version"`
	InstanceType  types.String `tfsdk:"instance_type"`
	Nodes         types.Int64  `tfsdk:"nodes"`
	Status        types.String `tfsdk:"status"`
	Region        types.String `tfsdk:"region"`
	ProjectID     types.String `tfsdk:"project_id"`
	CreatedAt     types.String `tfsdk:"created_at"`
}

func (r *QueueResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	data := cstypes.ConfigureResource(req, resp)
	if data != nil {
		r.client = data.V2Client
		r.accountID = data.AccountID
	}
}

func known(v types.String) bool { return !v.IsNull() && !v.IsUnknown() && v.ValueString() != "" }

// ValidateConfig mirrors the closed create body so mistakes surface at plan
// time instead of as a 400 on apply.
func (r *QueueResource) ValidateConfig(ctx context.Context, req resource.ValidateConfigRequest, resp *resource.ValidateConfigResponse) {
	var c queueResourceModel
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
	check("engine", c.Engine, validEngines, "rabbitmq")
	check("version", c.Version, validVersions, "4.3")
}

// apply maps a v2 Broker into Terraform state.
func (m *queueResourceModel) apply(c *apiQueue) {
	m.ID = types.StringValue(c.ID)
	m.Name = types.StringValue(c.Name)
	m.EnvironmentID = types.StringValue(c.EnvironmentID)
	m.Engine = types.StringValue(c.Engine)
	m.Version = types.StringPointerValue(c.Version)
	m.InstanceType = types.StringPointerValue(c.InstanceType)
	m.Nodes = types.Int64Value(c.Nodes)
	m.Status = types.StringValue(c.Status)
	m.Region = types.StringValue(c.Region)
	m.ProjectID = types.StringValue(c.ProjectID)
	m.CreatedAt = types.StringValue(c.CreatedAt)
}

func (r *QueueResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan queueResourceModel
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

	body := createQueueRequest{
		Name:          plan.Name.ValueString(),
		EnvironmentID: plan.EnvironmentID.ValueString(),
		Engine:        plan.Engine.ValueString(),
		InstanceType:  plan.InstanceType.ValueString(),
	}
	if known(plan.Version) {
		body.Version = cstypes.StringPtr(plan.Version.ValueString())
	}

	var created apiQueue
	status, raw, err := cstypes.DoJSON(ctx, r.client, http.MethodPost, "/queues", nil, body, &created)
	if err != nil {
		resp.Diagnostics.AddError("Error creating broker", err.Error())
		return
	}
	if status != http.StatusCreated {
		resp.Diagnostics.AddError("Error creating broker", cstypes.ProblemSummary(raw, status))
		return
	}

	state := plan
	state.apply(&created)
	// Save the id first so a failed wait doesn't orphan the broker.
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	if c := r.waitAvailable(ctx, created.ID, &resp.Diagnostics); c != nil {
		state.apply(c)
		resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
	}
}

// get reads a broker; gone is true on 404.
func (r *QueueResource) get(ctx context.Context, id string) (*apiQueue, bool, error) {
	var c apiQueue
	status, raw, err := cstypes.DoJSON(ctx, r.client, http.MethodGet, "/queues/"+url.PathEscape(id), nil, nil, &c)
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

// waitAvailable polls until the broker is available. It returns the broker,
// or nil with a diagnostic when it fails or the wait times out.
func (r *QueueResource) waitAvailable(ctx context.Context, id string, diags *diag.Diagnostics) *apiQueue {
	var last *apiQueue
	ok := dataproduct.Wait(ctx, "broker", waitTimeout, pollInterval, readyStatuses, failedStatuses,
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

func (r *QueueResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state queueResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	c, gone, err := r.get(ctx, state.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Error reading broker", err.Error())
		return
	}
	if gone {
		resp.State.RemoveResource(ctx)
		return
	}
	if c.ReadOnly {
		resp.Diagnostics.AddError("Broker is managed from its service",
			"This RabbitMQ broker was created as a service before SphereQueue instance types, so it can't be "+
				"managed with computesphere_queue. Manage it from its service instead.")
		return
	}
	state.apply(c)
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *QueueResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan, state queueResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	id := state.ID.ValueString()

	var body updateQueueRequest
	changed := false
	if known(plan.InstanceType) && plan.InstanceType.ValueString() != state.InstanceType.ValueString() {
		body.InstanceType = cstypes.StringPtr(plan.InstanceType.ValueString())
		changed = true
	}

	newState := plan
	if changed {
		var updated apiQueue
		status, raw, err := cstypes.DoJSON(ctx, r.client, http.MethodPatch, "/queues/"+url.PathEscape(id), nil, body, &updated)
		if err != nil {
			resp.Diagnostics.AddError("Error updating broker", err.Error())
			return
		}
		if status != http.StatusOK {
			resp.Diagnostics.AddError("Error updating broker", cstypes.ProblemSummary(raw, status))
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

func (r *QueueResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state queueResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	status, raw, err := cstypes.DoJSON(ctx, r.client, http.MethodDelete, "/queues/"+url.PathEscape(state.ID.ValueString()), nil, nil, nil)
	if err != nil {
		resp.Diagnostics.AddError("Error deleting broker", err.Error())
		return
	}
	switch status {
	case http.StatusOK, http.StatusAccepted, http.StatusNoContent, http.StatusNotFound:
		resp.State.RemoveResource(ctx)
	default:
		resp.Diagnostics.AddError("Error deleting broker", cstypes.ProblemSummary(raw, status))
	}
}

// ModifyPlan refuses a name, environment, engine or version change on an
// existing broker (no update body takes them)
// instead of planning a replacement that would delete its data.
func (r *QueueResource) ModifyPlan(ctx context.Context, req resource.ModifyPlanRequest, resp *resource.ModifyPlanResponse) {
	dataproduct.RefuseChange(ctx, req, resp, "broker", "name", "environment_id", "engine", "version")
}

func (r *QueueResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

func (r *QueueResource) IdentitySchema(ctx context.Context, req resource.IdentitySchemaRequest, resp *resource.IdentitySchemaResponse) {
	resp.IdentitySchema = identityschema.Schema{
		Attributes: map[string]identityschema.Attribute{
			"id": identityschema.StringAttribute{RequiredForImport: true},
		},
	}
}
