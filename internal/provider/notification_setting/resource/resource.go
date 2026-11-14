package provider

import (
	"context"
	"net/http"

	csv2 "github.com/computesphere/computesphere-go"
	cstypes "github.com/computesphere/terraform-provider-computesphere/internal/provider/types"
	openapi_types "github.com/oapi-codegen/runtime/types"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/identityschema"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
)

// webhookAttrTypes is the object type of one entry in webhooks.
var webhookAttrTypes = map[string]attr.Type{
	"url":          types.StringType,
	"content_type": types.StringType,
	"token":        types.StringType,
}

type webhookModel struct {
	URL         types.String `tfsdk:"url"`
	ContentType types.String `tfsdk:"content_type"`
	Token       types.String `tfsdk:"token"`
}

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
	Activity       types.Bool   `tfsdk:"activity"`
	Billing        types.Bool   `tfsdk:"billing"`
	Deployment     types.Bool   `tfsdk:"deployment"`
	EmailEnabled   types.Bool   `tfsdk:"email_enabled"`
	Emails         types.List   `tfsdk:"emails"`
	InappEnabled   types.Bool   `tfsdk:"inapp_enabled"`
	Invites        types.Bool   `tfsdk:"invites"`
	Payment        types.Bool   `tfsdk:"payment"`
	WebhookEnabled types.Bool   `tfsdk:"webhook_enabled"`
	Webhooks       types.List   `tfsdk:"webhooks"`
	ID             types.String `tfsdk:"id"`
	UserID         types.String `tfsdk:"user_id"`
}

// apply maps the API's settings into state. The API never returns a webhook
// token, so the token prior holds for the same URL (the plan or the state)
// is kept; prior may be null or unknown.
func (m *notificationSettingResourceModel) apply(ctx context.Context, n *csv2.NotificationSettings, prior types.List, diags *diag.Diagnostics) {
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
	emails := make([]string, 0, len(n.Emails))
	for _, e := range n.Emails {
		emails = append(emails, string(e))
	}
	list, d := types.ListValueFrom(ctx, types.StringType, emails)
	diags.Append(d...)
	m.Emails = list

	if prior.IsNull() || prior.IsUnknown() {
		// webhooks isn't managed here: leave it out of state.
		m.Webhooks = types.ListNull(basetypes.ObjectType{AttrTypes: webhookAttrTypes})
		return
	}
	tokens := map[string]types.String{}
	{
		var old []webhookModel
		diags.Append(prior.ElementsAs(ctx, &old, false)...)
		for _, w := range old {
			tokens[w.URL.ValueString()] = w.Token
		}
	}
	hooks := make([]webhookModel, 0, len(n.Webhooks))
	for _, w := range n.Webhooks {
		tok, ok := tokens[w.Url]
		if !ok {
			tok = types.StringNull()
		}
		hooks = append(hooks, webhookModel{URL: types.StringValue(w.Url), ContentType: types.StringValue(string(w.ContentType)), Token: tok})
	}
	hookList, hd := types.ListValueFrom(ctx, basetypes.ObjectType{AttrTypes: webhookAttrTypes}, hooks)
	diags.Append(hd...)
	m.Webhooks = hookList
}

func (m *notificationSettingResourceModel) toUpsert(emails []openapi_types.Email, hooks []csv2.NotificationWebhook) csv2.UpsertNotificationSettingsRequest {
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
		Webhooks:       hooks,
	}
}

// current reads the saved settings, for the lists the configuration leaves
// out. nil with no error means there are none yet.
func (r *NotificationSettingResource) current(ctx context.Context, diags *diag.Diagnostics) *csv2.NotificationSettings {
	cur, err := r.client.GetNotificationSettingsWithResponse(ctx)
	if err != nil {
		diags.AddError("Error reading notification setting", err.Error())
		return nil
	}
	switch {
	case cur.StatusCode() == http.StatusNotFound:
		return nil
	case cur.StatusCode() != http.StatusOK || cur.JSON200 == nil:
		diags.AddError("Error reading notification setting", cstypes.NotificationProblem(cur.Body, cur.StatusCode()))
		return nil
	}
	return cur.JSON200
}

// listsToSend are the emails and webhooks for an upsert, which replaces the
// saved ones. A configured list is sent as configured (a webhook token left
// out is sent blank, which keeps the stored token for that URL). A list the
// configuration leaves out is read and sent back unchanged, so what was set
// in the console stays.
func (r *NotificationSettingResource) listsToSend(ctx context.Context, plan notificationSettingResourceModel, diags *diag.Diagnostics) ([]openapi_types.Email, []csv2.NotificationWebhook) {
	emailsSet := !plan.Emails.IsNull() && !plan.Emails.IsUnknown()
	hooksSet := !plan.Webhooks.IsNull() && !plan.Webhooks.IsUnknown()
	var cur *csv2.NotificationSettings
	if !emailsSet || !hooksSet {
		if cur = r.current(ctx, diags); diags.HasError() {
			return nil, nil
		}
	}

	emails := []openapi_types.Email{}
	if emailsSet {
		var list []string
		diags.Append(plan.Emails.ElementsAs(ctx, &list, false)...)
		for _, e := range list {
			emails = append(emails, openapi_types.Email(e))
		}
	} else if cur != nil {
		emails = append(emails, cur.Emails...)
	}

	hooks := []csv2.NotificationWebhook{}
	if hooksSet {
		var list []webhookModel
		diags.Append(plan.Webhooks.ElementsAs(ctx, &list, false)...)
		for _, w := range list {
			ct := w.ContentType.ValueString()
			if ct == "" {
				ct = "json"
			}
			hooks = append(hooks, csv2.NotificationWebhook{Url: w.URL.ValueString(), ContentType: csv2.NotificationWebhookContentType(ct), Token: w.Token.ValueString()})
		}
	} else if cur != nil {
		for _, w := range cur.Webhooks {
			hooks = append(hooks, csv2.NotificationWebhook{Url: w.Url, ContentType: w.ContentType, Token: ""})
		}
	}
	return emails, hooks
}

func (r *NotificationSettingResource) upsert(ctx context.Context, plan notificationSettingResourceModel, diags *diag.Diagnostics) (*notificationSettingResourceModel, bool) {
	emails, hooks := r.listsToSend(ctx, plan, diags)
	if diags.HasError() {
		return nil, false
	}
	apiResp, err := r.client.UpsertNotificationSettingsWithResponse(ctx, plan.toUpsert(emails, hooks))
	if err != nil {
		diags.AddError("Error saving notification setting", err.Error())
		return nil, false
	}
	if apiResp.StatusCode() != http.StatusOK || apiResp.JSON200 == nil {
		diags.AddError("Error saving notification setting", cstypes.NotificationProblem(apiResp.Body, apiResp.StatusCode()))
		return nil, false
	}
	var state notificationSettingResourceModel
	state.apply(ctx, apiResp.JSON200, plan.Webhooks, diags)
	return &state, !diags.HasError()
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
		resp.Diagnostics.AddError("Error reading notification setting", cstypes.NotificationProblem(apiResp.Body, apiResp.StatusCode()))
		return
	}
	state.apply(ctx, apiResp.JSON200, state.Webhooks, &resp.Diagnostics)
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
