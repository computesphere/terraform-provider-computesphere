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
