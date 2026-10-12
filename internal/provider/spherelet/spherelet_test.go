package spherelet

import (
	"encoding/json"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/types"
)

func TestIsAlias(t *testing.T) {
	list := []Item{{ID: "sl-1c-2g", Aliases: []string{"MAX"}}, {ID: "sl-1c-4g"}}
	cases := []struct {
		configured, code string
		items            []Item
		want             bool
	}{
		{"sl-1c-2g", "sl-1c-2g", nil, true},
		{"max", "sl-1c-2g", list, true},
		{"MAX", "sl-1c-2g", nil, true},
		{"Flx", "sl-0.25c-512m", nil, true},
		{"pwr", "sl-1c-2g", nil, false},
		{"MAX", "sl-1c-4g", list, false},
		// A code the list doesn't carry falls back to the static map.
		{"PWR", "sl-2c-4g", list, true},
	}
	for _, c := range cases {
		if got := IsAlias(c.configured, c.code, c.items); got != c.want {
			t.Errorf("IsAlias(%q, %q) = %v, want %v", c.configured, c.code, got, c.want)
		}
	}
}

func TestValue(t *testing.T) {
	code := "sl-1c-2g"
	none := func() []Item { return nil }
	if got := Value(types.StringValue("MAX"), &code, true, none); got.ValueString() != "MAX" {
		t.Errorf("alias kept: got %v", got)
	}
	if got := Value(types.StringValue("sl-2c-4g"), &code, true, none); got.ValueString() != code {
		t.Errorf("drift shown: got %v", got)
	}
	if got := Value(types.StringUnknown(), &code, true, none); got.ValueString() != code {
		t.Errorf("unknown takes the API value: got %v", got)
	}
	if got := Value(types.StringValue("x"), nil, true, none); !got.IsNull() {
		t.Errorf("null from the API: got %v", got)
	}
	if got := Value(types.StringValue("x"), nil, false, none); got.ValueString() != "x" {
		t.Errorf("absent keeps prior: got %v", got)
	}
	if got := Value(types.StringUnknown(), nil, false, none); !got.IsNull() {
		t.Errorf("absent and unknown is null: got %v", got)
	}
}

func TestReadFields(t *testing.T) {
	f := ReadFields([]byte(`{"spherelet_type":"sl-1c-2g","plan_id":null}`))
	if !f.SphereletTypePresent || *f.SphereletType != "sl-1c-2g" || !f.PlanIDPresent || f.PlanID != nil {
		t.Errorf("service fields: %+v", f)
	}
	f = ReadDeploymentFields([]byte(`{"id":"d","raw":{"spherelet_type":"sl-2c-4g"}}`))
	if !f.SphereletTypePresent || *f.SphereletType != "sl-2c-4g" || f.PlanIDPresent {
		t.Errorf("deployment fields: %+v", f)
	}
	if f = ReadDeploymentFields([]byte(`{"id":"d"}`)); f.SphereletTypePresent {
		t.Errorf("no raw record: %+v", f)
	}
}

func TestSetFields(t *testing.T) {
	b, err := SetFields(struct {
		Name string `json:"name"`
	}{"svc"}, map[string]string{FieldType: "sl-1c-2g", FieldPlanID: ""})
	if err != nil {
		t.Fatal(err)
	}
	var m map[string]any
	if err := json.Unmarshal(b, &m); err != nil {
		t.Fatal(err)
	}
	if m["name"] != "svc" || m[FieldType] != "sl-1c-2g" {
		t.Errorf("body: %s", b)
	}
	if _, ok := m[FieldPlanID]; ok {
		t.Errorf("empty plan_id sent: %s", b)
	}
}
