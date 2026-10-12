package provider

import (
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
)

// git_repo_id is on every compute kind (all build from git); it never
// forces replacement (a change to a recorded value is refused at plan).
func TestGitRepoID(t *testing.T) {
	for _, k := range kinds {
		if !k.has("git_repo_id") {
			t.Errorf("%s: git_repo_id presence wrong", k.typeName)
		}
		if k.forcesReplace("git_repo_id") {
			t.Errorf("%s: git_repo_id must never force replacement", k.typeName)
		}
	}
}

// No compute resource exposes a server-only field or a plan override.
func TestSchemas_noServerOnlyFields(t *testing.T) {
	for _, k := range kinds {
		s := Schema(k)
		for _, f := range []string{"tag", "is_build_complete", "webhook_id", "pipeline_tier", "plan", "plan_id",
			"custom_domain", "db_config", "ci_config", "type"} {
			if _, ok := s.Attributes[f]; ok {
				t.Errorf("%s exposes %s", k.typeName, f)
			}
		}
	}
}

// Every attribute a kind lists has a definition, and every attribute that
// forces replacement is one the kind has.
func TestKinds_consistent(t *testing.T) {
	for _, k := range kinds {
		for _, a := range k.attrs {
			if _, ok := attrDefs[a]; !ok {
				t.Errorf("%s lists %s, which has no definition", k.typeName, a)
			}
		}
		for _, a := range k.replace {
			if !k.has(a) {
				t.Errorf("%s replaces on %s, which it doesn't have", k.typeName, a)
			}
		}
		if len(Schema(k).Attributes) != len(commonAttrs)+len(k.attrs) {
			t.Errorf("%s: schema has %d attributes, want %d", k.typeName, len(Schema(k).Attributes), len(commonAttrs)+len(k.attrs))
		}
	}
}

// A private service has no public address of any kind: no url, domain or
// network access attribute. Its only address is the internal one.
func TestPrivateService_noPublicAddress(t *testing.T) {
	// git_url and image_url are sources, not addresses of the service.
	sources := map[string]bool{"git_url": true, "image_url": true, "internal_url": true}
	for name := range Schema(privateService).Attributes {
		if sources[name] {
			continue
		}
		for _, bad := range []string{"url", "domain", "network", "address", "host"} {
			if strings.Contains(name, bad) {
				t.Errorf("computesphere_private_service exposes %s", name)
			}
		}
	}
}

// The registry password is write-only and never forces replacement.
func TestImagePassword_writeOnly(t *testing.T) {
	for _, k := range kinds {
		if !k.has("image_password") {
			continue
		}
		a, ok := Schema(k).Attributes["image_password"].(schema.StringAttribute)
		if !ok || !a.WriteOnly || !a.Sensitive {
			t.Errorf("%s: image_password must be sensitive and write-only", k.typeName)
		}
		if k.forcesReplace("image_password") || k.forcesReplace("image_password_wo_version") {
			t.Errorf("%s: a password rotation must not replace the resource", k.typeName)
		}
	}
}
