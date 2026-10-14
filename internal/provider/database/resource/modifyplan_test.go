package provider

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/stretchr/testify/assert"
)

func TestThreeZoneRequested(t *testing.T) {
	assert.True(t, threeZoneRequested(types.StringValue("three_zone"), types.StringNull()))
	assert.True(t, threeZoneRequested(types.StringNull(), types.StringValue("ha")))
	assert.False(t, threeZoneRequested(types.StringValue("standby"), types.StringNull()))
	assert.False(t, threeZoneRequested(types.StringNull(), types.StringValue("standard")))
	assert.False(t, threeZoneRequested(types.StringUnknown(), types.StringNull()))
}
