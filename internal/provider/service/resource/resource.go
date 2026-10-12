package provider

import (
	"bytes"
	"context"
	"net/http"
	"time"

	csv2 "github.com/computesphere/computesphere-go"
	"github.com/computesphere/terraform-provider-computesphere/internal/provider/spherelet"
	cstypes "github.com/computesphere/terraform-provider-computesphere/internal/provider/types"
	"github.com/google/uuid"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/identityschema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

type ServiceResource struct {
	client    *csv2.ClientWithResponses
	accountID string
}

var _ resource.Resource = &ServiceResource{}
var _ resource.ResourceWithIdentity = &ServiceResource{}
var _ resource.ResourceWithValidateConfig = &ServiceResource{}

func NewServiceResource() resource.Resource {
	return &ServiceResource{}
}

func (r *ServiceResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = "computesphere_service"
}

func (r *ServiceResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = Schema(ctx)
}

func (r *ServiceResource) IdentitySchema(ctx context.Context, req resource.IdentitySchemaRequest, resp *resource.IdentitySchemaResponse) {
	resp.IdentitySchema = identityschema.Schema{
		Attributes: map[string]identityschema.Attribute{
			"id": identityschema.StringAttribute{RequiredForImport: true},
		},
	}
}

// serviceResourceModel — env_id/image_name/port/sphere_count are intentionally
// absent: under the compute model those are deployment-level concerns, not part
// of the v2 service resource. A service carries its spherelet type.
type serviceResourceModel struct {
	Name          types.String `tfsdk:"name"`
	ProjectID     types.String `tfsdk:"project_id"`
	Type          types.String `tfsdk:"type"`
	SphereletType types.String `tfsdk:"spherelet_type"`
	PlanID        types.String `tfsdk:"plan_id"`
	Active        types.Bool   `tfsdk:"active"`
	CreatedAt     types.String `tfsdk:"created_at"`
	LastOpenedAt  types.String `tfsdk:"last_opened_at"`
	ID            types.String `tfsdk:"id"`
}

func (r *ServiceResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	data := cstypes.ConfigureResource(req, resp)
	if data != nil {
		r.client = data.V2Client
		r.accountID = data.AccountID
	}
}

// ValidateConfig rejects spherelet_type and the deprecated plan_id together
// at plan time.
func (r *ServiceResource) ValidateConfig(ctx context.Context, req resource.ValidateConfigRequest, resp *resource.ValidateConfigResponse) {
	var c serviceResourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &c)...)
	if resp.Diagnostics.HasError() {
		return
	}
	if !c.SphereletType.IsNull() && !c.PlanID.IsNull() {
		resp.Diagnostics.AddAttributeError(path.Root(spherelet.FieldPlanID), "Conflicting attributes", spherelet.ConflictDetail)
	}
}

// known returns the value of a planned attribute, or "" when it is null or
// unknown (not set in the configuration).
func known(v types.String) string {
	if v.IsNull() || v.IsUnknown() {
		return ""
	}
	return v.ValueString()
}

// apply maps a v2 service into state. body is the raw response, read for the
// spherelet type; prior is the planned or prior-state value, so an alias in
// the configuration is kept as written.
func (m *serviceResourceModel) apply(s *csv2.Service, body []byte, prior types.String, aliases func() []spherelet.Item) {
	f := spherelet.ReadFields(body)
	m.ID = types.StringValue(s.Id)
	m.Name = types.StringValue(s.Name)
	m.ProjectID = types.StringValue(s.ProjectId.String())
	m.Type = types.StringValue(s.Type)
	m.SphereletType = spherelet.Value(prior, f.SphereletType, f.SphereletTypePresent, aliases)
	m.PlanID = types.StringPointerValue(s.PlanId)
	m.Active = types.BoolValue(s.Active)
	m.CreatedAt = types.StringValue(s.CreatedAt.Format(time.RFC3339))
	m.LastOpenedAt = cstypes.TimePtrString(s.LastOpenedAt)
}

func (r *ServiceResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan serviceResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	projectID, err := uuid.Parse(plan.ProjectID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Invalid project_id", err.Error())
		return
	}

	body, err := spherelet.SetFields(csv2.CreateServiceRequest{
		Name:      plan.Name.ValueString(),
		ProjectId: projectID,
		Type:      plan.Type.ValueString(),
		PlanId:    cstypes.StringPtr(known(plan.PlanID)),
	}, map[string]string{spherelet.FieldType: known(plan.SphereletType)})
	if err != nil {
		resp.Diagnostics.AddError("Error creating service", err.Error())
		return
	}
	apiResp, err := r.client.CreateServiceWithBodyWithResponse(ctx, "application/json", bytes.NewReader(body))
	if err != nil {
		resp.Diagnostics.AddError("Error creating service", err.Error())
		return
	}
	if apiResp.StatusCode() != http.StatusCreated || apiResp.JSON201 == nil {
		resp.Diagnostics.AddError("Error creating service", cstypes.ProblemSummary(apiResp.Body, apiResp.StatusCode()))
		return
	}

	var state serviceResourceModel
	state.apply(apiResp.JSON201, apiResp.Body, plan.SphereletType, spherelet.AliasLookup(ctx, r.client))
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *ServiceResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state serviceResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	sid, err := uuid.Parse(state.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Invalid service id", err.Error())
		return
	}

	apiResp, err := r.client.GetServiceWithResponse(ctx, sid)
	if err != nil {
		resp.Diagnostics.AddError("Error reading service", err.Error())
		return
	}
	if apiResp.StatusCode() == http.StatusNotFound {
		resp.State.RemoveResource(ctx)
		return
	}
	if apiResp.StatusCode() != http.StatusOK || apiResp.JSON200 == nil {
		resp.Diagnostics.AddError("Error reading service", cstypes.ProblemSummary(apiResp.Body, apiResp.StatusCode()))
		return
	}

	state.apply(apiResp.JSON200, apiResp.Body, state.SphereletType, spherelet.AliasLookup(ctx, r.client))
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *ServiceResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan serviceResourceModel
	var state serviceResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	sid, err := uuid.Parse(state.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Invalid service id", err.Error())
		return
	}

	// The v2 service update surface is the spherelet type (or the deprecated
	// plan_id); name/type/project changes force replacement via
	// RequiresReplace. Only attributes set in the configuration are sent: an
	// unset one is unknown in the plan.
	fields := map[string]string{
		spherelet.FieldType:   known(plan.SphereletType),
		spherelet.FieldPlanID: known(plan.PlanID),
	}
	var (
		svc     *csv2.Service
		rawBody []byte
		status  int
	)
	if fields[spherelet.FieldType] == "" && fields[spherelet.FieldPlanID] == "" {
		// Nothing updatable changed; refresh from the API.
		getResp, gerr := r.client.GetServiceWithResponse(ctx, sid)
		if gerr != nil {
			resp.Diagnostics.AddError("Error updating service", gerr.Error())
			return
		}
		svc, rawBody, status = getResp.JSON200, getResp.Body, getResp.StatusCode()
	} else {
		body, berr := spherelet.SetFields(struct{}{}, fields)
		if berr != nil {
			resp.Diagnostics.AddError("Error updating service", berr.Error())
			return
		}
		apiResp, uerr := r.client.UpdateServiceWithBodyWithResponse(ctx, sid, "application/json", bytes.NewReader(body))
		if uerr != nil {
			resp.Diagnostics.AddError("Error updating service", uerr.Error())
			return
		}
		svc, rawBody, status = apiResp.JSON200, apiResp.Body, apiResp.StatusCode()
	}
	if status != http.StatusOK || svc == nil {
		resp.Diagnostics.AddError("Error updating service", cstypes.ProblemSummary(rawBody, status))
		return
	}

	var newState serviceResourceModel
	newState.apply(svc, rawBody, plan.SphereletType, spherelet.AliasLookup(ctx, r.client))
	resp.Diagnostics.Append(resp.State.Set(ctx, &newState)...)
}

func (r *ServiceResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state serviceResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	sid, err := uuid.Parse(state.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Invalid service id", err.Error())
		return
	}

	apiResp, err := r.client.DeleteServiceWithResponse(ctx, sid)
	if err != nil {
		resp.Diagnostics.AddError("Error deleting service", err.Error())
		return
	}
	switch apiResp.StatusCode() {
	case http.StatusOK, http.StatusNoContent, http.StatusAccepted, http.StatusNotFound:
		resp.State.RemoveResource(ctx)
	default:
		resp.Diagnostics.AddError("Error deleting service", cstypes.ProblemSummary(apiResp.Body, apiResp.StatusCode()))
	}
}
