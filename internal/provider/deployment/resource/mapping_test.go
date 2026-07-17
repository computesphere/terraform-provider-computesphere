package provider

import (
	"context"
	"testing"

	csv2 "github.com/computesphere/computesphere-go"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// mustStringMap builds a types.Map of string for test input.
func mustStringMap(t *testing.T, in map[string]string) types.Map {
	t.Helper()
	m, diags := types.MapValueFrom(context.Background(), types.StringType, in)
	if diags.HasError() {
		t.Fatalf("failed to build map value: %v", diags)
	}
	return m
}

func TestBuildUpdateBody_PrivateImageAndBuildArgs(t *testing.T) {
	ctx := context.Background()
	var diags diag.Diagnostics

	plan := deploymentResourceModel{
		Image:         types.StringValue("myregistry.azurecr.io/app:1.2.3"),
		ImageType:     types.StringValue("private"),
		ImageProvider: types.StringValue("azure"),
		ImageUsername: types.StringValue("svc-user"),
		ImagePassword: types.StringValue("super-secret-token"),
		ImageURL:      types.StringValue("myregistry.azurecr.io"),
		BuildArgs:     mustStringMap(t, map[string]string{"NEXT_PUBLIC_API_URL": "https://api.example.com"}),
		Port:          types.Int64Value(8080),
	}

	upd := buildUpdateBody(ctx, plan, &diags)
	if diags.HasError() {
		t.Fatalf("unexpected diagnostics: %v", diags)
	}

	if upd.Image == nil {
		t.Fatal("expected Image to be set")
	}
	if got := deref(upd.Image.Name); got != "myregistry.azurecr.io/app:1.2.3" {
		t.Errorf("Image.Name = %q", got)
	}
	if upd.Image.Type == nil || *upd.Image.Type != csv2.DeploymentUpdateRequestImageTypePrivate {
		t.Errorf("Image.Type = %v, want private", upd.Image.Type)
	}
	if got := deref(upd.Image.Provider); got != "azure" {
		t.Errorf("Image.Provider = %q", got)
	}
	if got := deref(upd.Image.Username); got != "svc-user" {
		t.Errorf("Image.Username = %q", got)
	}
	if got := deref(upd.Image.Password); got != "super-secret-token" {
		t.Errorf("Image.Password = %q", got)
	}
	if got := deref(upd.Image.Url); got != "myregistry.azurecr.io" {
		t.Errorf("Image.Url = %q", got)
	}

	if upd.BuildConfig == nil || upd.BuildConfig.BuildArgs == nil {
		t.Fatal("expected BuildConfig.BuildArgs to be set")
	}
	if got := (*upd.BuildConfig.BuildArgs)["NEXT_PUBLIC_API_URL"]; got != "https://api.example.com" {
		t.Errorf("BuildArgs[NEXT_PUBLIC_API_URL] = %q", got)
	}

	if upd.Port == nil || *upd.Port != 8080 {
		t.Errorf("Port = %v, want 8080", upd.Port)
	}
}

func TestBuildUpdateBody_PublicImageNoAuth(t *testing.T) {
	ctx := context.Background()
	var diags diag.Diagnostics

	plan := deploymentResourceModel{
		Image:         types.StringValue("nginx:latest"),
		ImageType:     types.StringNull(),
		ImageProvider: types.StringNull(),
		ImageUsername: types.StringNull(),
		ImagePassword: types.StringNull(),
		ImageURL:      types.StringNull(),
		BuildArgs:     types.MapNull(types.StringType),
		Port:          types.Int64Null(),
	}

	upd := buildUpdateBody(ctx, plan, &diags)
	if diags.HasError() {
		t.Fatalf("unexpected diagnostics: %v", diags)
	}

	if upd.Image == nil {
		t.Fatal("expected Image to be set (name only)")
	}
	if got := deref(upd.Image.Name); got != "nginx:latest" {
		t.Errorf("Image.Name = %q", got)
	}
	if upd.Image.Type != nil || upd.Image.Provider != nil || upd.Image.Username != nil ||
		upd.Image.Password != nil || upd.Image.Url != nil {
		t.Errorf("expected no auth fields set for a public image, got %+v", upd.Image)
	}
	if upd.BuildConfig != nil {
		t.Errorf("expected no BuildConfig when build_args unset, got %+v", upd.BuildConfig)
	}
	if upd.Port != nil {
		t.Errorf("expected Port unset, got %v", *upd.Port)
	}
}

func TestBuildUpdateBody_Empty(t *testing.T) {
	ctx := context.Background()
	var diags diag.Diagnostics

	plan := deploymentResourceModel{
		Image:         types.StringNull(),
		ImageType:     types.StringNull(),
		ImageProvider: types.StringNull(),
		ImageUsername: types.StringNull(),
		ImagePassword: types.StringNull(),
		ImageURL:      types.StringNull(),
		BuildArgs:     types.MapNull(types.StringType),
		Port:          types.Int64Null(),
	}

	upd := buildUpdateBody(ctx, plan, &diags)
	if diags.HasError() {
		t.Fatalf("unexpected diagnostics: %v", diags)
	}
	if upd.Image != nil {
		t.Errorf("expected Image unset for an empty plan, got %+v", upd.Image)
	}
	if upd.BuildConfig != nil {
		t.Errorf("expected BuildConfig unset, got %+v", upd.BuildConfig)
	}

	if deploymentNeedsUpdate(ctx, plan, &diags) {
		t.Error("deploymentNeedsUpdate should be false for an empty plan")
	}
}

func TestDeploymentNeedsUpdate(t *testing.T) {
	ctx := context.Background()
	var diags diag.Diagnostics

	base := func() deploymentResourceModel {
		return deploymentResourceModel{
			Image:         types.StringValue("nginx:latest"),
			ImageType:     types.StringNull(),
			ImageProvider: types.StringNull(),
			ImageUsername: types.StringNull(),
			ImagePassword: types.StringNull(),
			ImageURL:      types.StringNull(),
			BuildArgs:     types.MapNull(types.StringType),
		}
	}

	// Flat public image alone must NOT trigger the extra update call.
	if deploymentNeedsUpdate(ctx, base(), &diags) {
		t.Error("public flat image should not need an update call")
	}

	p := base()
	p.ImagePassword = types.StringValue("tok")
	if !deploymentNeedsUpdate(ctx, p, &diags) {
		t.Error("private-image password should require an update call")
	}

	p = base()
	p.BuildArgs = mustStringMap(t, map[string]string{"FOO": "bar"})
	if !deploymentNeedsUpdate(ctx, p, &diags) {
		t.Error("build_args should require an update call")
	}
}

func deref(p *string) string {
	if p == nil {
		return ""
	}
	return *p
}
