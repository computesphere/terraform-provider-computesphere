// Package dataproduct holds what the SphereCache and SphereQueue resources
// share: the name rule of their typed creates and the status poll that waits
// for a create or resize to settle.
package dataproduct

import (
	"context"
	"fmt"
	"regexp"
	"time"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// RefuseChange refuses, at plan time, a change to any of attrs on an
// existing resource: a data product's name, environment, engine and version
// can't change in place, and replacing it would delete its data, so the plan stops with an
// error instead of planning a replacement. noun names the resource, e.g.
// "cache".
func RefuseChange(ctx context.Context, req resource.ModifyPlanRequest, resp *resource.ModifyPlanResponse, noun string, attrs ...string) {
	if req.State.Raw.IsNull() || req.Plan.Raw.IsNull() {
		return
	}
	for _, attr := range attrs {
		var planV, stateV types.String
		resp.Diagnostics.Append(req.Plan.GetAttribute(ctx, path.Root(attr), &planV)...)
		resp.Diagnostics.Append(req.State.GetAttribute(ctx, path.Root(attr), &stateV)...)
		if resp.Diagnostics.HasError() {
			return
		}
		if planV.IsUnknown() || planV.IsNull() || stateV.IsNull() || planV.Equal(stateV) {
			continue
		}
		resp.Diagnostics.AddAttributeError(path.Root(attr), fmt.Sprintf("%s can't change on an existing %s", attr, noun),
			fmt.Sprintf("This %s has %s %s, and it can't be changed in place. Changing it needs a new %s, which "+
				"would start empty: create the new %s as a separate resource, move your data, then remove this one. "+
				"To keep this %s, set %s back to %q.",
				noun, attr, stateV.ValueString(), noun, noun, noun, attr, stateV.ValueString()))
	}
}

// NameRe is the name rule of the typed data-product creates: lowercase
// letters, digits and hyphens, starting with a letter, 1-40 characters.
var NameRe = regexp.MustCompile(`^[a-z]([a-z0-9-]*[a-z0-9])?$`)

// ValidName reports whether name satisfies the typed create's name rule.
func ValidName(name string) bool {
	return len(name) >= 1 && len(name) <= 40 && NameRe.MatchString(name)
}

// Snapshot is one status read taken while waiting.
type Snapshot struct {
	Status string
	Error  string // customer-safe error text; empty when there is none
	Gone   bool   // the resource answered 404
}

// Wait polls get until the status is in ready, the status is in failed, the
// resource disappears, ctx is cancelled or timeout passes. It returns true
// when a ready status was reached and adds an error diagnostic otherwise.
// noun names the resource in diagnostics, e.g. "cache".
func Wait(ctx context.Context, noun string, timeout, interval time.Duration,
	ready, failed map[string]bool, get func(context.Context) (Snapshot, error), diags *diag.Diagnostics) bool {
	deadline := time.Now().Add(timeout)
	for {
		s, err := get(ctx)
		if err != nil {
			diags.AddError("Error waiting for "+noun, err.Error())
			return false
		}
		if s.Gone {
			diags.AddError(fmt.Sprintf("The %s disappeared", noun),
				fmt.Sprintf("the %s was deleted while waiting for it to become available", noun))
			return false
		}
		if ready[s.Status] {
			return true
		}
		if failed[s.Status] {
			msg := s.Error
			if msg == "" {
				msg = fmt.Sprintf("the %s reached status %s", noun, s.Status)
			}
			diags.AddError(fmt.Sprintf("The %s failed", noun), msg)
			return false
		}
		if time.Now().After(deadline) {
			diags.AddError("Timed out waiting for "+noun,
				fmt.Sprintf("the %s is still %s after %s", noun, s.Status, timeout))
			return false
		}
		select {
		case <-ctx.Done():
			diags.AddError("Cancelled waiting for "+noun, ctx.Err().Error())
			return false
		case <-time.After(interval):
		}
	}
}
