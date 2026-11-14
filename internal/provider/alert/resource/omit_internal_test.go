package provider

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/types"
)

func TestOmitUnset(t *testing.T) {
	body := func(plan alertResourceModel) map[string]any {
		req, _ := http.NewRequest(http.MethodPost, "http://x/v2/alerts", bytes.NewReader([]byte(`{"alert_type":"job_run_failed","threshold":0,"evaluation_period":0,"severity_level":"high"}`)))
		if err := omitUnset(plan)(context.Background(), req); err != nil {
			t.Fatal(err)
		}
		raw, _ := io.ReadAll(req.Body)
		out := map[string]any{}
		_ = json.Unmarshal(raw, &out)
		return out
	}
	unset := body(alertResourceModel{Threshold: types.Int64Null(), EvaluationPeriod: types.Int64Unknown()})
	if _, ok := unset["threshold"]; ok {
		t.Errorf("threshold sent while unset: %v", unset)
	}
	if _, ok := unset["evaluation_period"]; ok {
		t.Errorf("evaluation_period sent while unset: %v", unset)
	}
	if unset["alert_type"] != "job_run_failed" {
		t.Errorf("other fields lost: %v", unset)
	}
	both := body(alertResourceModel{Threshold: types.Int64Value(3), EvaluationPeriod: types.Int64Value(120)})
	if _, ok := both["threshold"]; !ok {
		t.Errorf("threshold dropped while set: %v", both)
	}
}
