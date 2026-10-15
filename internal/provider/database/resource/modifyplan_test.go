package provider

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/stretchr/testify/assert"
)

func TestThreeZoneRequested(t *testing.T) {
	assert.True(t, threeZoneRequested(types.StringValue("three_zone")))
	assert.False(t, threeZoneRequested(types.StringValue("standby")))
	assert.False(t, threeZoneRequested(types.StringNull()))
	assert.False(t, threeZoneRequested(types.StringUnknown()))
}

func TestEngineVersionError(t *testing.T) {
	for _, c := range []struct {
		engine, version string
		bad             bool
	}{
		{"documentdb", "18", false},
		{"documentdb", "17", true},
		{"postgresql", "16", false},
		{"postgresql", "18", false},
	} {
		if got := engineVersionError(c.engine, c.version) != ""; got != c.bad {
			t.Errorf("engineVersionError(%q, %q) refuses = %v, want %v", c.engine, c.version, got, c.bad)
		}
	}
}
