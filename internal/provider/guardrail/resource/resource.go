package provider

import (
	"context"

	cs "github.com/computesphere/cli/cs"
	cstypes "github.com/computesphere/terraform-provider-computesphere/internal/provider/types"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/identityschema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

type GuardrailResource struct {
	client *cs.APIClient
}

var _ resource.Resource = &GuardrailResource{}
var _ resource.ResourceWithIdentity = &GuardrailResource{}

func NewGuardrailResource() resource.Resource {
	return &GuardrailResource{}
}

func (r *GuardrailResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = "computesphere_guardrail"
}

func (r *GuardrailResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = Schema(ctx)
}

func (r *GuardrailResource) IdentitySchema(ctx context.Context, req resource.IdentitySchemaRequest, resp *resource.IdentitySchemaResponse) {
	resp.IdentitySchema = identityschema.Schema{
		Attributes: map[string]identityschema.Attribute{
			"id": identityschema.StringAttribute{RequiredForImport: true},
		},
	}
}

type guardrailResourceModel struct {
	Name                 types.String              `tfsdk:"name"`
	Description          types.String              `tfsdk:"description"`
	Effect               types.String              `tfsdk:"effect"`
	Message              types.String              `tfsdk:"message"`
	Rules                []map[string]types.String `tfsdk:"rules"`
	Scope                types.String              `tfsdk:"scope"`
	Status               types.Bool                `tfsdk:"status"`
	Type                 types.String              `tfsdk:"type"`
	ID                   types.String              `tfsdk:"id"`
	AccountID            types.String              `tfsdk:"account_id"`
	CreatedBy            types.String              `tfsdk:"created_by"`
	IsPredefinedAssigned types.Bool                `tfsdk:"is_predefined_assigned"`
}

func (r *GuardrailResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	data := cstypes.ConfigureResource(req, resp)
	if data != nil {
		r.client = data.Client
	}
}

func (r *GuardrailResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan guardrailResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	var statusPtr *bool
	if !plan.Status.IsNull() {
		b := plan.Status.ValueBool()
		statusPtr = &b
	}
	// Convert rules from []map[string]types.String to []string
	var rules []string
	if plan.Rules != nil {
		for _, ruleMap := range plan.Rules {
			// Flatten the map to a string representation (e.g., JSON or key=value pairs)
			// Here, we use a simple key=value; for more complex rules, adjust as needed
			for k, v := range ruleMap {
				rules = append(rules, k+"="+v.ValueString())
			}
		}
	}
	payload := cs.ModelGuardrailPayload{
		Name:        ptrString(plan.Name.ValueString()),
		Description: ptrString(plan.Description.ValueString()),
		Effect:      ptrString(plan.Effect.ValueString()),
		Message:     ptrString(plan.Message.ValueString()),
		Scope:       ptrString(plan.Scope.ValueString()),
		Status:      statusPtr,
		Type:        ptrString(plan.Type.ValueString()),
		Rules:       rules,
	}
	_, _, err := r.client.GuardrailsAPI.GuardrailsPost(ctx).Body(payload).Execute()
	if err != nil {
		resp.Diagnostics.AddError("Error creating guardrail", err.Error())
		return
	}
	// After creation, fetch the guardrail by name to get the ID
	// (Assume a List API or similar exists; if not, this may need to be adjusted)
	// For now, set state as best effort
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *GuardrailResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state guardrailResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	apiResp, httpResp, err := r.client.GuardrailsAPI.GuardrailsIdGet(ctx, state.ID.ValueString()).Execute()
	if err != nil {
		if httpResp != nil && httpResp.StatusCode == 404 {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Error reading guardrail", err.Error())
		return
	}
	if apiResp.Data == nil {
		resp.State.RemoveResource(ctx)
		return
	}
	g := apiResp.Data
	state.ID = types.StringValue(g.GetId())
	state.Name = types.StringValue(g.GetName())
	state.Description = types.StringValue(g.GetDescription())
	state.Effect = types.StringValue(g.GetEffect())
	state.Message = types.StringValue(g.GetMessage())
	state.Scope = types.StringValue(g.GetScope())
	state.Status = types.BoolValue(g.GetStatus())
	state.Type = types.StringValue(g.GetType())
	state.AccountID = types.StringValue(g.GetAccountId())
	state.CreatedBy = types.StringValue(g.GetCreatedBy())
	state.IsPredefinedAssigned = types.BoolValue(g.GetIsPredefinedAssigned())
	// Convert rules from []ModelGuardrailSingleRuleResponse to []map[string]types.String
	if g.Rules != nil {
		rules := make([]map[string]types.String, 0, len(g.Rules))
		for _, rule := range g.Rules {
			m := map[string]types.String{}
			if rule.Name != nil {
				m["name"] = types.StringValue(*rule.Name)
			}
			if rule.Operator != nil {
				m["operator"] = types.StringValue(*rule.Operator)
			}
			if rule.Scope != nil {
				m["scope"] = types.StringValue(*rule.Scope)
			}
			if rule.Metrics != nil {
				m["metrics"] = types.StringValue(*rule.Metrics)
			}
			// Value is a map[string]interface{}, convert to string if needed
			if rule.Value != nil {
				m["value"] = types.StringValue("<complex>") // Or serialize as needed
			}
			rules = append(rules, m)
		}
		state.Rules = rules
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *GuardrailResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan guardrailResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	var statusPtr *bool
	if !plan.Status.IsNull() {
		b := plan.Status.ValueBool()
		statusPtr = &b
	}
	// Convert rules from []map[string]types.String to []string
	var rules []string
	if plan.Rules != nil {
		for _, ruleMap := range plan.Rules {
			for k, v := range ruleMap {
				rules = append(rules, k+"="+v.ValueString())
			}
		}
	}
	payload := cs.ModelGuardrailPayload{
		Name:        ptrString(plan.Name.ValueString()),
		Description: ptrString(plan.Description.ValueString()),
		Effect:      ptrString(plan.Effect.ValueString()),
		Message:     ptrString(plan.Message.ValueString()),
		Scope:       ptrString(plan.Scope.ValueString()),
		Status:      statusPtr,
		Type:        ptrString(plan.Type.ValueString()),
		Rules:       rules,
	}
	_, _, err := r.client.GuardrailsAPI.GuardrailsIdPut(ctx, plan.ID.ValueString()).Body(payload).Execute()
	if err != nil {
		resp.Diagnostics.AddError("Error updating guardrail", err.Error())
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *GuardrailResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state guardrailResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	_, httpResp, err := r.client.GuardrailsAPI.GuardrailsIdDelete(ctx, state.ID.ValueString()).Execute()
	if err != nil {
		if httpResp != nil && httpResp.StatusCode == 404 {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Error deleting guardrail", err.Error())
		return
	}
	resp.State.RemoveResource(ctx)
}

// ptrString returns a pointer to the given string.
func ptrString(s string) *string {
	return &s
}
