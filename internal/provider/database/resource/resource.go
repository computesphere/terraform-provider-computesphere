package provider

import (
	"context"
	"fmt"
	"net/http"
	"time"

	csv2 "github.com/computesphere/computesphere-go"
	cstypes "github.com/computesphere/terraform-provider-computesphere/internal/provider/types"
	"github.com/google/uuid"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/identityschema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// waitTimeout bounds how long create and resize wait for the database to
// become available (provisioning a new instance takes a few minutes).
const waitTimeout = 30 * time.Minute

// pollInterval is how often the database status is read while waiting.
var pollInterval = 10 * time.Second

// SetPollIntervalForTest shortens the status poll so replayed acceptance
// tests don't sleep between recorded polls. Test use only.
func SetPollIntervalForTest(d time.Duration) { pollInterval = d }

type DatabaseResource struct {
	client    *csv2.ClientWithResponses
	accountID string
}

var _ resource.Resource = &DatabaseResource{}
var _ resource.ResourceWithImportState = &DatabaseResource{}
var _ resource.ResourceWithIdentity = &DatabaseResource{}
var _ resource.ResourceWithValidateConfig = &DatabaseResource{}

func NewDatabaseResource() resource.Resource {
	return &DatabaseResource{}
}

func (r *DatabaseResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = "computesphere_database"
}

func (r *DatabaseResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = Schema(ctx)
}

type databaseResourceModel struct {
	ID               types.String `tfsdk:"id"`
	Name             types.String `tfsdk:"name"`
	EnvironmentID    types.String `tfsdk:"environment_id"`
	Engine           types.String `tfsdk:"engine"`
	Version          types.String `tfsdk:"version"`
	InstanceType     types.String `tfsdk:"instance_type"`
	HighAvailability types.String `tfsdk:"high_availability"`
	Plan             types.String `tfsdk:"plan"`
	StorageGB        types.Int64  `tfsdk:"storage_gb"`
	Inject           types.Bool   `tfsdk:"inject"`
	Status           types.String `tfsdk:"status"`
	Region           types.String `tfsdk:"region"`
	ProjectID        types.String `tfsdk:"project_id"`
	MinorVersion     types.String `tfsdk:"minor_version"`
	CreatedAt        types.String `tfsdk:"created_at"`
}

func (r *DatabaseResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	data := cstypes.ConfigureResource(req, resp)
	if data != nil {
		r.client = data.V2Client
		r.accountID = data.AccountID
	}
}

var (
	validEngines  = map[string]bool{"postgresql": true}
	validVersions = map[string]bool{"16": true, "17": true, "18": true}
	validHA       = map[string]bool{"none": true, "standby": true, "three_zone": true}
	validPlans    = map[string]bool{"hobby": true, "standard": true, "ha": true}
)

// ValidateConfig mirrors the API's create rules so mistakes surface at plan
// time: exactly one of instance_type or the deprecated plan, plan can't be
// combined with high_availability, and enum values are checked.
func (r *DatabaseResource) ValidateConfig(ctx context.Context, req resource.ValidateConfigRequest, resp *resource.ValidateConfigResponse) {
	var c databaseResourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &c)...)
	if resp.Diagnostics.HasError() {
		return
	}
	set := func(v types.String) bool { return !v.IsNull() && !v.IsUnknown() }
	if set(c.InstanceType) && set(c.Plan) {
		resp.Diagnostics.AddAttributeError(path.Root("plan"), "Conflicting attributes",
			"Set instance_type (and optionally high_availability) or the deprecated plan, not both.")
	}
	if c.InstanceType.IsNull() && c.Plan.IsNull() {
		resp.Diagnostics.AddAttributeError(path.Root("instance_type"), "Missing instance type",
			"Set instance_type, e.g. \"pg-2c-4g\". List the options with the computesphere_database_instance_types data source.")
	}
	if set(c.Plan) && set(c.HighAvailability) {
		resp.Diagnostics.AddAttributeError(path.Root("high_availability"), "Conflicting attributes",
			"high_availability can't be combined with the deprecated plan; use instance_type.")
	}
	check := func(attr string, v types.String, ok map[string]bool, list string) {
		if set(v) && !ok[v.ValueString()] {
			resp.Diagnostics.AddAttributeError(path.Root(attr), "Invalid value", fmt.Sprintf("%s must be one of %s.", attr, list))
		}
	}
	check("engine", c.Engine, validEngines, "postgresql")
	check("version", c.Version, validVersions, "16, 17, 18")
	check("high_availability", c.HighAvailability, validHA, "none, standby, three_zone")
	check("plan", c.Plan, validPlans, "hobby, standard, ha")
	if !c.StorageGB.IsNull() && !c.StorageGB.IsUnknown() && (c.StorageGB.ValueInt64() < 1 || c.StorageGB.ValueInt64() > 2048) {
		resp.Diagnostics.AddAttributeError(path.Root("storage_gb"), "Invalid value", "storage_gb must be between 1 and 2048.")
	}
}

// apply maps a v2 Database into Terraform state. inject is create-only and
// isn't returned by the API, so the configured value is kept.
func (m *databaseResourceModel) apply(d *csv2.Database) {
	m.ID = types.StringValue(d.Id.String())
	m.Name = types.StringValue(d.Name)
	m.EnvironmentID = types.StringValue(d.EnvironmentId.String())
	m.Engine = types.StringValue(string(d.Engine))
	m.Version = types.StringValue(d.Version)
	m.InstanceType = types.StringValue(d.InstanceType)
	m.HighAvailability = types.StringValue(string(d.HighAvailability))
	m.Plan = types.StringValue(string(d.Plan))
	m.StorageGB = types.Int64Value(int64(d.StorageGb))
	m.Status = types.StringValue(string(d.Status))
	m.Region = types.StringValue(d.Region)
	m.ProjectID = types.StringValue(d.ProjectId.String())
	m.MinorVersion = types.StringPointerValue(d.MinorVersion)
	m.CreatedAt = types.StringValue(d.CreatedAt.Format(time.RFC3339))
	if m.Inject.IsNull() || m.Inject.IsUnknown() {
		m.Inject = types.BoolValue(true)
	}
}

func (r *DatabaseResource) accountUUID(diags *diag.Diagnostics) (uuid.UUID, bool) {
	id, err := uuid.Parse(r.accountID)
	if err != nil {
		diags.AddError("Invalid account_id", err.Error())
		return uuid.Nil, false
	}
	return id, true
}

func (r *DatabaseResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan databaseResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	accountID, ok := r.accountUUID(&resp.Diagnostics)
	if !ok {
		return
	}
	envID, err := uuid.Parse(plan.EnvironmentID.ValueString())
	if err != nil {
		resp.Diagnostics.AddAttributeError(path.Root("environment_id"), "Invalid environment_id", err.Error())
		return
	}

	body := csv2.CreateDatabaseRequest{
		Name:          plan.Name.ValueString(),
		EnvironmentId: envID,
		Engine:        csv2.DatabaseEngine(plan.Engine.ValueString()),
	}
	if known(plan.Version) {
		v := csv2.CreateDatabaseRequestVersion(plan.Version.ValueString())
		body.Version = &v
	}
	if known(plan.InstanceType) {
		body.InstanceType = cstypes.StringPtr(plan.InstanceType.ValueString())
	}
	if known(plan.HighAvailability) {
		ha := csv2.DatabaseHighAvailability(plan.HighAvailability.ValueString())
		body.HighAvailability = &ha
	}
	if known(plan.Plan) {
		p := csv2.DatabasePlanCode(plan.Plan.ValueString())
		body.Plan = &p
	}
	if !plan.StorageGB.IsNull() && !plan.StorageGB.IsUnknown() {
		gb := int(plan.StorageGB.ValueInt64())
		body.StorageGb = &gb
	}
	inject := plan.Inject.ValueBool()
	body.Inject = &inject

	apiResp, err := r.client.CreateDatabaseWithResponse(ctx, &csv2.CreateDatabaseParams{XAccountId: accountID}, body)
	if err != nil {
		resp.Diagnostics.AddError("Error creating database", err.Error())
		return
	}
	if apiResp.StatusCode() != http.StatusCreated || apiResp.JSON201 == nil {
		resp.Diagnostics.AddError("Error creating database", cstypes.ProblemSummary(apiResp.Body, apiResp.StatusCode()))
		return
	}

	state := plan
	state.apply(apiResp.JSON201)
	// Save the id first so a failed wait doesn't orphan the database.
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	if db := r.waitAvailable(ctx, apiResp.JSON201.Id, &resp.Diagnostics); db != nil {
		state.apply(db)
		resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
	}
}

func known(v types.String) bool { return !v.IsNull() && !v.IsUnknown() && v.ValueString() != "" }

// waitAvailable polls until the database is available (or degraded:
// serving, with a standby still catching up). It returns the database, or
// nil with a diagnostic when it fails or the wait times out.
func (r *DatabaseResource) waitAvailable(ctx context.Context, id uuid.UUID, diags *diag.Diagnostics) *csv2.Database {
	deadline := time.Now().Add(waitTimeout)
	for {
		db, gone, err := r.get(ctx, id)
		if err != nil {
			diags.AddError("Error waiting for database", err.Error())
			return nil
		}
		if gone {
			diags.AddError("Database disappeared", "the database was deleted while waiting for it to become available")
			return nil
		}
		switch db.Status {
		case csv2.DatabaseStatusAvailable, csv2.DatabaseStatusDegraded:
			return db
		case csv2.DatabaseStatusFailed:
			msg := db.Error
			if msg == "" {
				msg = "the database reached status failed"
			}
			diags.AddError("Database failed", msg)
			return nil
		}
		if time.Now().After(deadline) {
			diags.AddError("Timed out waiting for database",
				fmt.Sprintf("the database is still %s after %s", db.Status, waitTimeout))
			return nil
		}
		select {
		case <-ctx.Done():
			diags.AddError("Cancelled waiting for database", ctx.Err().Error())
			return nil
		case <-time.After(pollInterval):
		}
	}
}

// get reads a database; gone is true on 404.
func (r *DatabaseResource) get(ctx context.Context, id uuid.UUID) (*csv2.Database, bool, error) {
	apiResp, err := r.client.GetDatabaseWithResponse(ctx, id)
	if err != nil {
		return nil, false, err
	}
	if apiResp.StatusCode() == http.StatusNotFound {
		return nil, true, nil
	}
	if apiResp.StatusCode() != http.StatusOK || apiResp.JSON200 == nil {
		return nil, false, fmt.Errorf("%s", cstypes.ProblemSummary(apiResp.Body, apiResp.StatusCode()))
	}
	return apiResp.JSON200, false, nil
}

func (r *DatabaseResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state databaseResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	id, err := uuid.Parse(state.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Invalid database id", err.Error())
		return
	}
	db, gone, err := r.get(ctx, id)
	if err != nil {
		resp.Diagnostics.AddError("Error reading database", err.Error())
		return
	}
	if gone {
		resp.State.RemoveResource(ctx)
		return
	}
	state.apply(db)
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *DatabaseResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan, state databaseResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	id, err := uuid.Parse(state.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Invalid database id", err.Error())
		return
	}

	var body csv2.UpdateDatabaseRequest
	changed := false
	if known(plan.InstanceType) && plan.InstanceType.ValueString() != state.InstanceType.ValueString() {
		body.InstanceType = cstypes.StringPtr(plan.InstanceType.ValueString())
		changed = true
	}
	if known(plan.HighAvailability) && plan.HighAvailability.ValueString() != state.HighAvailability.ValueString() {
		ha := csv2.DatabaseHighAvailability(plan.HighAvailability.ValueString())
		body.HighAvailability = &ha
		changed = true
	}
	if known(plan.Plan) && body.InstanceType == nil && body.HighAvailability == nil &&
		plan.Plan.ValueString() != state.Plan.ValueString() {
		p := csv2.DatabasePlanCode(plan.Plan.ValueString())
		body.Plan = &p
		changed = true
	}
	if !plan.StorageGB.IsNull() && !plan.StorageGB.IsUnknown() && plan.StorageGB.ValueInt64() != state.StorageGB.ValueInt64() {
		if plan.StorageGB.ValueInt64() < state.StorageGB.ValueInt64() {
			resp.Diagnostics.AddAttributeError(path.Root("storage_gb"), "Storage can only grow",
				fmt.Sprintf("storage_gb can't go from %d to %d.", state.StorageGB.ValueInt64(), plan.StorageGB.ValueInt64()))
			return
		}
		gb := int(plan.StorageGB.ValueInt64())
		body.StorageGb = &gb
		changed = true
	}

	newState := plan
	if changed {
		apiResp, err := r.client.UpdateDatabaseWithResponse(ctx, id, body)
		if err != nil {
			resp.Diagnostics.AddError("Error updating database", err.Error())
			return
		}
		if apiResp.StatusCode() != http.StatusOK || apiResp.JSON200 == nil {
			resp.Diagnostics.AddError("Error updating database", cstypes.ProblemSummary(apiResp.Body, apiResp.StatusCode()))
			return
		}
		newState.apply(apiResp.JSON200)
		if db := r.waitAvailable(ctx, id, &resp.Diagnostics); db != nil {
			newState.apply(db)
		}
	} else if db, _, err := r.get(ctx, id); err == nil && db != nil {
		newState.apply(db)
	}
	if !plan.Inject.Equal(state.Inject) {
		resp.Diagnostics.AddAttributeWarning(path.Root("inject"), "inject only applies at create",
			"Changing inject doesn't change an existing database's connection-string injection.")
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &newState)...)
}

func (r *DatabaseResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state databaseResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	id, err := uuid.Parse(state.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Invalid database id", err.Error())
		return
	}
	apiResp, err := r.client.DeleteDatabaseWithResponse(ctx, id)
	if err != nil {
		resp.Diagnostics.AddError("Error deleting database", err.Error())
		return
	}
	switch apiResp.StatusCode() {
	case http.StatusOK, http.StatusAccepted, http.StatusNoContent, http.StatusNotFound:
		resp.State.RemoveResource(ctx)
	default:
		resp.Diagnostics.AddError("Error deleting database", cstypes.ProblemSummary(apiResp.Body, apiResp.StatusCode()))
	}
}

func (r *DatabaseResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

func (r *DatabaseResource) IdentitySchema(ctx context.Context, req resource.IdentitySchemaRequest, resp *resource.IdentitySchemaResponse) {
	resp.IdentitySchema = identityschema.Schema{
		Attributes: map[string]identityschema.Attribute{
			"id": identityschema.StringAttribute{RequiredForImport: true},
		},
	}
}
