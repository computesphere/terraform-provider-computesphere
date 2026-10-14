package provider

import (
	"context"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestWholeSetAddsReplace(t *testing.T) {
	req, err := http.NewRequest(http.MethodPut, "https://api.example.com/v2/deployments/x/env-vars?a=1", nil)
	require.NoError(t, err)
	require.NoError(t, wholeSet(context.Background(), req))
	assert.Equal(t, "true", req.URL.Query().Get("replace"))
	assert.Equal(t, "1", req.URL.Query().Get("a"))
}
