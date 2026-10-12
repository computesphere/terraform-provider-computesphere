package provider

import "testing"

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
