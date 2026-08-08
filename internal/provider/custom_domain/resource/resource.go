package provider

import (
	"context"
	"net/http"

	csv2 "github.com/computesphere/computesphere-go"
	cstypes "github.com/computesphere/terraform-provider-computesphere/internal/provider/types"
	"github.com/google/uuid"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

type CustomDomainResource struct {
	client    *csv2.ClientWithResponses
	accountID string
}

var _ resource.Resource = &CustomDomainResource{}
var _ resource.ResourceWithImportState = &CustomDomainResource{}

func NewCustomDomainResource() resource.Resource {
	return &CustomDomainResource{}
}

func (r *CustomDomainResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = "computesphere_custom_domain"
}

func (r *CustomDomainResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = Schema(ctx)
}

// customDomainResourceModel exposes only the server-issued id and hostname. The
// internal dom-<hash> slug is never mapped into state (ADR 0132 guardrail).
type customDomainResourceModel struct {
	ID           types.String `tfsdk:"id"`
	DeploymentID types.String `tfsdk:"deployment_id"`
	Domain       types.String `tfsdk:"domain"`
	Hostname     types.String `tfsdk:"hostname"`
	Status       types.String `tfsdk:"status"`
	Verified     types.Bool   `tfsdk:"verified"`
}

func (r *CustomDomainResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	data := cstypes.ConfigureResource(req, resp)
	if data != nil {
		r.client = data.V2Client
		r.accountID = data.AccountID
	}
}

// apply maps the server Domain object into state. deployment_id and the
// user-supplied domain input are preserved by the caller; this only touches the
// server-computed attributes. The server id and hostname are surfaced; the
// internal slug is never referenced.
func (m *customDomainResourceModel) apply(d *csv2.Domain) {
	if d.Id != nil {
		m.ID = types.StringValue(*d.Id)
	}
	// Hostname is an alias of domain; fall back to the domain field when the
	// server omits the alias.
	if d.Hostname != nil {
		m.Hostname = types.StringValue(*d.Hostname)
	} else {
		m.Hostname = types.StringValue(d.Domain)
	}
	if d.Status != nil {
		m.Status = types.StringValue(string(*d.Status))
		m.Verified = types.BoolValue(*d.Status == csv2.DomainStatusActive)
	} else {
		m.Status = types.StringNull()
		m.Verified = types.BoolValue(false)
	}
}

func (r *CustomDomainResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan customDomainResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	deploymentID, err := uuid.Parse(plan.DeploymentID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Invalid deployment_id", err.Error())
		return
	}

	apiResp, err := r.client.AddDeploymentDomainWithResponse(ctx, deploymentID, csv2.AddDomainRequest{
		Domain: plan.Domain.ValueString(),
	})
	if err != nil {
		resp.Diagnostics.AddError("Error adding custom domain", err.Error())
		return
	}
	if apiResp.StatusCode() != http.StatusCreated || apiResp.JSON201 == nil {
		resp.Diagnostics.AddError("Error adding custom domain", cstypes.ProblemSummary(apiResp.Body, apiResp.StatusCode()))
		return
	}

	state := plan
	state.apply(apiResp.JSON201)
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *CustomDomainResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state customDomainResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	deploymentID, err := uuid.Parse(state.DeploymentID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Invalid deployment_id", err.Error())
		return
	}

	apiResp, err := r.client.GetDeploymentDomainWithResponse(ctx, deploymentID, state.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Error reading custom domain", err.Error())
		return
	}
	if apiResp.StatusCode() == http.StatusNotFound {
		resp.State.RemoveResource(ctx)
		return
	}
	if apiResp.StatusCode() != http.StatusOK || apiResp.JSON200 == nil {
		resp.Diagnostics.AddError("Error reading custom domain", cstypes.ProblemSummary(apiResp.Body, apiResp.StatusCode()))
		return
	}

	state.apply(apiResp.JSON200)
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *CustomDomainResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	// Both deployment_id and domain force replacement, so every attribute is
	// immutable; update only carries state forward.
	var plan customDomainResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *CustomDomainResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state customDomainResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	deploymentID, err := uuid.Parse(state.DeploymentID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Invalid deployment_id", err.Error())
		return
	}

	apiResp, err := r.client.DeleteDeploymentDomainWithResponse(ctx, deploymentID, state.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Error deleting custom domain", err.Error())
		return
	}
	switch apiResp.StatusCode() {
	case http.StatusOK, http.StatusNoContent, http.StatusAccepted, http.StatusNotFound:
		resp.State.RemoveResource(ctx)
	default:
		resp.Diagnostics.AddError("Error deleting custom domain", cstypes.ProblemSummary(apiResp.Body, apiResp.StatusCode()))
	}
}

// ImportState accepts "<deployment_id>/<domain_id>" so both identifiers needed
// by the deployment-scoped domain endpoints are populated.
func (r *CustomDomainResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	id, deploymentID, ok := splitImportID(req.ID)
	if !ok {
		resp.Diagnostics.AddError(
			"Invalid import ID",
			"Expected import ID in the format \"<deployment_id>/<domain_id>\", got: "+req.ID,
		)
		return
	}
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("deployment_id"), deploymentID)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), id)...)
}

// splitImportID parses "<deployment_id>/<domain_id>" into (domainID, deploymentID).
func splitImportID(s string) (domainID, deploymentID string, ok bool) {
	for i := 0; i < len(s); i++ {
		if s[i] == '/' {
			deploymentID, domainID = s[:i], s[i+1:]
			if deploymentID == "" || domainID == "" {
				return "", "", false
			}
			return domainID, deploymentID, true
		}
	}
	return "", "", false
}
