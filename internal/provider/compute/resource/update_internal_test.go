package provider

import (
	"context"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// Update refuses a git_repo_id that differs from the recorded one before
// any API call: the resource has no client here, so reaching the API would
// panic.
func TestUpdate_refusesGitRepoIDChangeBeforeAPI(t *testing.T) {
	ctx := context.Background()
	r := &computeResource{kind: webService}
	s := Schema(webService)
	base, d := fromObject(ctx, types.ObjectNull(nil))
	if d.HasError() {
		t.Fatal(d)
	}
	base.ID = types.StringValue("00000000-0000-4000-8000-000000000001")
	base.ServiceID = types.StringValue("44444444-4444-4444-4444-444444444444")
	base.EnvironmentID = types.StringValue("33333333-3333-3333-3333-333333333333")
	base.ProjectID = types.StringValue("22222222-2222-2222-2222-222222222222")
	base.WaitForReady = types.BoolValue(true)
	base.GitURL = types.StringValue("group/app")

	raw := func(repoID string) tfsdk.State {
		m := base
		m.GitRepoID = types.StringValue(repoID)
		obj, d := toObject(ctx, webService, m)
		if d.HasError() {
			t.Fatal(d)
		}
		v, err := obj.ToTerraformValue(ctx)
		if err != nil {
			t.Fatal(err)
		}
		return tfsdk.State{Schema: s, Raw: v}
	}
	state, planned := raw("4821"), raw("9999")
	req := resource.UpdateRequest{
		State:  state,
		Plan:   tfsdk.Plan{Schema: s, Raw: planned.Raw},
		Config: tfsdk.Config{Schema: s, Raw: planned.Raw},
	}
	resp := &resource.UpdateResponse{State: state}
	r.Update(ctx, req, resp)
	if !resp.Diagnostics.HasError() || !strings.Contains(resp.Diagnostics.Errors()[0].Summary(), "git_repo_id can't change") {
		t.Fatalf("diagnostics = %v, want the git_repo_id refusal", resp.Diagnostics)
	}
}
