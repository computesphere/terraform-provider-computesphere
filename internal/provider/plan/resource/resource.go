package provider

import (
	"context"

	cs "github.com/computesphere/cli/cs"
	cstypes "github.com/computesphere/terraform-provider-computesphere/internal/provider/types"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/identityschema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

type PlanResource struct {
	client *cs.APIClient
}

var _ resource.Resource = &PlanResource{}
var _ resource.ResourceWithIdentity = &PlanResource{}

func NewPlanResource() resource.Resource {
	return &PlanResource{}
}

func (r *PlanResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = "computesphere_plan"
}

func (r *PlanResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = Schema(ctx)
}

func (r *PlanResource) IdentitySchema(ctx context.Context, req resource.IdentitySchemaRequest, resp *resource.IdentitySchemaResponse) {
	resp.IdentitySchema = identityschema.Schema{
		Attributes: map[string]identityschema.Attribute{
			"id": identityschema.StringAttribute{RequiredForImport: true},
		},
	}
}

type planResourceModel struct {
	ID           types.String  `tfsdk:"id"`
	Active       types.Bool    `tfsdk:"active"`
	Core         types.Int64   `tfsdk:"core"`
	CountryCode  types.String  `tfsdk:"country_code"`
	CurrencyCode types.String  `tfsdk:"currency_code"`
	Memory       types.Int64   `tfsdk:"memory"`
	Name         types.String  `tfsdk:"name"`
	Price        types.Float64 `tfsdk:"price"`
	Slug         types.String  `tfsdk:"slug"`
	Type         types.String  `tfsdk:"type"`
}

func (r *PlanResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	data := cstypes.ConfigureResource(req, resp)
	if data != nil {
		r.client = data.Client
	}
}

func (r *PlanResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan planResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
	readResp := &resource.ReadResponse{State: resp.State, Diagnostics: resp.Diagnostics}
	r.Read(ctx, resource.ReadRequest{State: resp.State}, readResp)
	resp.Diagnostics = readResp.Diagnostics
	resp.State = readResp.State
}

func (r *PlanResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state planResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	apiResp, httpResp, err := r.client.PlanAPI.PlansIdGet(ctx, state.ID.ValueString()).Execute()
	if err != nil {
		if httpResp != nil && httpResp.StatusCode == 404 {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Error reading plan", err.Error())
		return
	}
	if apiResp.Data == nil {
		resp.State.RemoveResource(ctx)
		return
	}
	plan := apiResp.Data
	state.Active = types.BoolValue(plan.GetActive())
	state.Core = types.Int64Value(int64(plan.GetCore()))
	state.CountryCode = types.StringValue(plan.GetCountryCode())
	state.CurrencyCode = types.StringValue(plan.GetCurrencyCode())
	state.Memory = types.Int64Value(int64(plan.GetMemory()))
	state.Name = types.StringValue(plan.GetName())
	state.Price = types.Float64Value(float64(plan.GetPrice()))
	state.Slug = types.StringValue(plan.GetSlug())
	state.Type = types.StringValue(plan.GetType())
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *PlanResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan planResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
	readResp := &resource.ReadResponse{State: resp.State, Diagnostics: resp.Diagnostics}
	r.Read(ctx, resource.ReadRequest{State: resp.State}, readResp)
	resp.Diagnostics = readResp.Diagnostics
	resp.State = readResp.State
}

func (r *PlanResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	resp.State.RemoveResource(ctx)
}
