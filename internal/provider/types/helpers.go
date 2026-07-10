package types

import (
	"net/http"
	"time"

	"github.com/hashicorp/terraform-plugin-framework/types"
)

// ProblemSummary formats the RFC 7807 Problem body returned by the v2 API for
// Terraform diagnostic output. It prefers the raw body; when empty, it falls
// back to the HTTP status text. Shared by every migrated domain so the error
// surface is uniform.
func ProblemSummary(body []byte, status int) string {
	if len(body) > 0 {
		return string(body)
	}
	return http.StatusText(status)
}

// StringPtr returns a pointer to s, or nil when s is empty. The v2 request
// bodies model optional string fields as *string (omitempty), so an empty
// Terraform value maps to "field omitted" rather than "set to empty string".
func StringPtr(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}

// TimePtrString maps an optional v2 timestamp (*time.Time) to a Terraform
// string value, or null when the timestamp is absent.
func TimePtrString(t *time.Time) types.String {
	if t == nil {
		return types.StringNull()
	}
	return types.StringValue(t.Format(time.RFC3339))
}
