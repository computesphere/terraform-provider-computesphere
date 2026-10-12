package provider

import "testing"

// git_repo_id is on every git-source kind except the static site, whose
// create body has no ci_config, and no change body takes it.
func TestGitRepoID(t *testing.T) {
	for _, k := range kinds {
		if k.has("git_repo_id") != !k.is(staticSite) {
			t.Errorf("%s: git_repo_id presence wrong", k.typeName)
		}
		if k.has("git_repo_id") && !k.forcesReplace("git_repo_id") {
			t.Errorf("%s: git_repo_id must force replacement", k.typeName)
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
