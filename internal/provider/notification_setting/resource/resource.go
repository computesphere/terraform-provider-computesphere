package provider

import (
	"context"
	"net/http"

	csv2 "github.com/computesphere/computesphere-go"
	cstypes "github.com/computesphere/terraform-provider-computesphere/internal/provider/types"
	openapi_types "github.com/oapi-codegen/runtime/types"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/identityschema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// NotificationSettingResource manages the account's singleton notification
// settings. The v2 API exposes get + upsert (no create/delete), so create and
// update both upsert, and delete only drops the resource from state.
type NotificationSettingResource struct {
	client    *csv2.ClientWithResponses
	accountID string
}

var _ resource.Resource = &NotificationSettingResource{}
var _ resource.ResourceWithIdentity = &NotificationSettingResource{}

func NewNotificationSettingResource() resource.Resource {
	return &NotificationSettingResource{}
}

func (r *NotificationSettingResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = "computesphere_notification_setting"
}

func (r *NotificationSettingResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = Schema(ctx)
}

func (r *NotificationSettingResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	data := cstypes.ConfigureResource(req, resp)
	if data != nil {
		r.client = data.V2Client
		r.accountID = data.AccountID
	}
}

func (r *NotificationSettingResource) IdentitySchema(ctx context.Context, req resource.IdentitySchemaRequest, resp *resource.IdentitySchemaResponse) {
	resp.IdentitySchema = identityschema.Schema{
		Attributes: map[string]identityschema.Attribute{
			"id": identityschema.StringAttribute{RequiredForImport: true},
		},
	}
}

type notificationSettingResourceModel struct {
	Activity       types.Bool     `tfsdk:"activity"`
	Billing        types.Bool     `tfsdk:"billing"`
	Deployment     types.Bool     `tfsdk:"deployment"`
	EmailEnabled   types.Bool     `tfsdk:"email_enabled"`
	Emails         []types.String `tfsdk:"emails"`
	InappEnabled   types.Bool     `tfsdk:"inapp_enabled"`
	Invites        types.Bool     `tfsdk:"invites"`
	Payment        types.Bool     `tfsdk:"payment"`
	WebhookEnabled types.Bool     `tfsdk:"webhook_enabled"`
	ID             types.String   `tfsdk:"id"`
	UserID         types.String   `tfsdk:"user_id"`
}

func (m *notificationSettingResourceModel) apply(n *csv2.NotificationSettings) {
	m.ID = types.StringValue(n.Id.String())
	m.UserID = types.StringValue(n.UserId.String())
	m.Activity = types.BoolValue(n.Activity)
	m.Billing = types.BoolValue(n.Billing)
	m.Deployment = types.BoolValue(n.Deployment)
	m.EmailEnabled = types.BoolValue(n.EmailEnabled)
	m.InappEnabled = types.BoolValue(n.InappEnabled)
	m.Invites = types.BoolValue(n.Invites)
	m.Payment = types.BoolValue(n.Payment)
	m.WebhookEnabled = types.BoolValue(n.WebhookEnabled)
	emails := make([]types.String, 0, len(n.Emails))
	for _, e := range n.Emails {
		emails = append(emails, types.StringValue(string(e)))
	}
	m.Emails = emails
}

func (m *notificationSettingResourceModel) toUpsert() csv2.UpsertNotificationSettingsRequest {
	emails := make([]openapi_types.Email, 0, len(m.Emails))
	for _, e := range m.Emails {
		emails = append(emails, openapi_types.Email(e.ValueString()))
	}
	return csv2.UpsertNotificationSettingsRequest{
		Activity:       m.Activity.ValueBool(),
		Billing:        m.Billing.ValueBool(),
		Deployment:     m.Deployment.ValueBool(),
		EmailEnabled:   m.EmailEnabled.ValueBool(),
		Emails:         emails,
		InappEnabled:   m.InappEnabled.ValueBool(),
		Invites:        m.Invites.ValueBool(),
		Payment:        m.Payment.ValueBool(),
		WebhookEnabled: m.WebhookEnabled.ValueBool(),
		Webhooks:       []csv2.NotificationWebhook{},
	}
}

func (r *NotificationSettingResource) upsert(ctx context.Context, plan notificationSettingResourceModel, diags interface {
	AddError(string, string)
}) (*notificationSettingResourceModel, bool) {
	apiResp, err := r.client.UpsertNotificationSettingsWithResponse(ctx, plan.toUpsert())
	if err != nil {
		diags.AddError("Error saving notification setting", err.Error())
		return nil, false
	}
	if apiResp.StatusCode() != http.StatusOK || apiResp.JSON200 == nil {
		diags.AddError("Error saving notification setting", cstypes.ProblemSummary(apiResp.Body, apiResp.StatusCode()))
		return nil, false
	}
	var state notificationSettingResourceModel
	state.apply(apiResp.JSON200)
	return &state, true
}

func (r *NotificationSettingResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan notificationSettingResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	state, ok := r.upsert(ctx, plan, &resp.Diagnostics)
	if !ok {
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, state)...)
}

func (r *NotificationSettingResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state notificationSettingResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	apiResp, err := r.client.GetNotificationSettingsWithResponse(ctx)
	if err != nil {
		resp.Diagnostics.AddError("Error reading notification setting", err.Error())
		return
	}
	if apiResp.StatusCode() == http.StatusNotFound {
		resp.State.RemoveResource(ctx)
		return
	}
	if apiResp.StatusCode() != http.StatusOK || apiResp.JSON200 == nil {
		resp.Diagnostics.AddError("Error reading notification setting", cstypes.ProblemSummary(apiResp.Body, apiResp.StatusCode()))
		return
	}
	state.apply(apiResp.JSON200)
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *NotificationSettingResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan notificationSettingResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	state, ok := r.upsert(ctx, plan, &resp.Diagnostics)
	if !ok {
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, state)...)
}

func (r *NotificationSettingResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	// No delete API for the singleton settings; drop it from state only.
	resp.State.RemoveResource(ctx)
}
