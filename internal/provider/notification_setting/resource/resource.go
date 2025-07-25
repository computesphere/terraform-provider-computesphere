package provider

import (
	"context"

	cs "github.com/computesphere/cli/cs"
	cstypes "github.com/computesphere/terraform-provider-computesphere/internal/provider/types"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/identityschema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

type NotificationSettingResource struct {
	client *cs.APIClient
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
		r.client = data.Client
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
	Activity       types.Bool                `tfsdk:"activity"`
	Billing        types.Bool                `tfsdk:"billing"`
	Deployment     types.Bool                `tfsdk:"deployment"`
	EmailEnabled   types.Bool                `tfsdk:"email_enabled"`
	Emails         []types.String            `tfsdk:"emails"`
	InappEnabled   types.Bool                `tfsdk:"inapp_enabled"`
	Invites        types.Bool                `tfsdk:"invites"`
	Payment        types.Bool                `tfsdk:"payment"`
	WebhookEnabled types.Bool                `tfsdk:"webhook_enabled"`
	Webhooks       []map[string]types.String `tfsdk:"webhooks"`
	ID             types.String              `tfsdk:"id"`
	UserID         types.String              `tfsdk:"user_id"`
}

func (r *NotificationSettingResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan notificationSettingResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	client := r.client
	payload := cs.ModelNotificationSettingRequestPayload{}
	if !plan.Activity.IsNull() {
		b := plan.Activity.ValueBool()
		payload.Activity = &b
	}
	if !plan.Billing.IsNull() {
		b := plan.Billing.ValueBool()
		payload.Billing = &b
	}
	if !plan.Deployment.IsNull() {
		b := plan.Deployment.ValueBool()
		payload.Deployment = &b
	}
	if !plan.EmailEnabled.IsNull() {
		b := plan.EmailEnabled.ValueBool()
		payload.EmailEnabled = &b
	}
	if plan.Emails != nil {
		emails := []string{}
		for _, e := range plan.Emails {
			if !e.IsNull() {
				emails = append(emails, e.ValueString())
			}
		}
		payload.Emails = emails
	}
	if !plan.InappEnabled.IsNull() {
		b := plan.InappEnabled.ValueBool()
		payload.InappEnabled = &b
	}
	if !plan.Invites.IsNull() {
		b := plan.Invites.ValueBool()
		payload.Invites = &b
	}
	if !plan.Payment.IsNull() {
		b := plan.Payment.ValueBool()
		payload.Payment = &b
	}
	if !plan.WebhookEnabled.IsNull() {
		b := plan.WebhookEnabled.ValueBool()
		payload.WebhookEnabled = &b
	}
	// Webhooks omitted for brevity; add as needed
	apiResp, _, err := client.NotificationAPI.NotificationsSettingsPut(ctx).Body(payload).Execute()
	if err != nil {
		resp.Diagnostics.AddError("Error creating notification setting", err.Error())
		return
	}
	if apiResp.Data == nil || apiResp.Data.Id == nil {
		resp.Diagnostics.AddError("Notification setting creation failed", "No ID returned")
		return
	}
	plan.ID = types.StringValue(*apiResp.Data.Id)
	plan.UserID = types.StringPointerValue(apiResp.Data.UserId)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *NotificationSettingResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state notificationSettingResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	client := r.client
	apiResp, httpResp, err := client.NotificationAPI.NotificationsSettingsGet(ctx).Execute()
	if err != nil {
		if httpResp != nil && httpResp.StatusCode == 404 {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Error reading notification setting", err.Error())
		return
	}
	if apiResp.Data == nil || apiResp.Data.Id == nil {
		resp.State.RemoveResource(ctx)
		return
	}
	state.ID = types.StringValue(apiResp.Data.GetId())
	if apiResp.Data.UserId != nil {
		state.UserID = types.StringPointerValue(apiResp.Data.UserId)
	} else {
		state.UserID = types.StringNull()
	}
	if apiResp.Data.Activity != nil {
		state.Activity = types.BoolPointerValue(apiResp.Data.Activity)
	} else {
		state.Activity = types.BoolNull()
	}
	if apiResp.Data.Billing != nil {
		state.Billing = types.BoolPointerValue(apiResp.Data.Billing)
	} else {
		state.Billing = types.BoolNull()
	}
	if apiResp.Data.Deployment != nil {
		state.Deployment = types.BoolPointerValue(apiResp.Data.Deployment)
	} else {
		state.Deployment = types.BoolNull()
	}
	if apiResp.Data.EmailEnabled != nil {
		state.EmailEnabled = types.BoolPointerValue(apiResp.Data.EmailEnabled)
	} else {
		state.EmailEnabled = types.BoolNull()
	}
	if apiResp.Data.Emails != nil {
		emails := make([]types.String, 0, len(apiResp.Data.Emails))
		for _, e := range apiResp.Data.Emails {
			emails = append(emails, types.StringValue(e))
		}
		state.Emails = emails
	}
	if apiResp.Data.InappEnabled != nil {
		state.InappEnabled = types.BoolPointerValue(apiResp.Data.InappEnabled)
	} else {
		state.InappEnabled = types.BoolNull()
	}
	if apiResp.Data.Invites != nil {
		state.Invites = types.BoolPointerValue(apiResp.Data.Invites)
	} else {
		state.Invites = types.BoolNull()
	}
	if apiResp.Data.Payment != nil {
		state.Payment = types.BoolPointerValue(apiResp.Data.Payment)
	} else {
		state.Payment = types.BoolNull()
	}
	if apiResp.Data.WebhookEnabled != nil {
		state.WebhookEnabled = types.BoolPointerValue(apiResp.Data.WebhookEnabled)
	} else {
		state.WebhookEnabled = types.BoolNull()
	}
	// Webhooks omitted for brevity; add as needed
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *NotificationSettingResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	// Use the same logic as Create
	var plan notificationSettingResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	client := r.client
	payload := cs.ModelNotificationSettingRequestPayload{}
	if !plan.Activity.IsNull() {
		b := plan.Activity.ValueBool()
		payload.Activity = &b
	}
	if !plan.Billing.IsNull() {
		b := plan.Billing.ValueBool()
		payload.Billing = &b
	}
	if !plan.Deployment.IsNull() {
		b := plan.Deployment.ValueBool()
		payload.Deployment = &b
	}
	if !plan.EmailEnabled.IsNull() {
		b := plan.EmailEnabled.ValueBool()
		payload.EmailEnabled = &b
	}
	if plan.Emails != nil {
		emails := []string{}
		for _, e := range plan.Emails {
			if !e.IsNull() {
				emails = append(emails, e.ValueString())
			}
		}
		payload.Emails = emails
	}
	if !plan.InappEnabled.IsNull() {
		b := plan.InappEnabled.ValueBool()
		payload.InappEnabled = &b
	}
	if !plan.Invites.IsNull() {
		b := plan.Invites.ValueBool()
		payload.Invites = &b
	}
	if !plan.Payment.IsNull() {
		b := plan.Payment.ValueBool()
		payload.Payment = &b
	}
	if !plan.WebhookEnabled.IsNull() {
		b := plan.WebhookEnabled.ValueBool()
		payload.WebhookEnabled = &b
	}
	// Webhooks omitted for brevity; add as needed
	apiResp, _, err := client.NotificationAPI.NotificationsSettingsPut(ctx).Body(payload).Execute()
	if err != nil {
		resp.Diagnostics.AddError("Error updating notification setting", err.Error())
		return
	}
	if apiResp.Data == nil || apiResp.Data.Id == nil {
		resp.Diagnostics.AddError("Notification setting update failed", "No ID returned")
		return
	}
	plan.ID = types.StringValue(*apiResp.Data.Id)
	plan.UserID = types.StringPointerValue(apiResp.Data.UserId)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *NotificationSettingResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	// No API to delete, so just remove from state
	resp.State.RemoveResource(ctx)
}
