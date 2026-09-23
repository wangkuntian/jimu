package main

import (
	"bytes"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRunPrintsProfileAssets(t *testing.T) {
	var out bytes.Buffer
	require.NoError(t, run([]string{"full"}, &out))
	assert.Contains(t, out.String(), "docs/openapi")
	assert.Contains(t, out.String(), "deploy/openobserve")
	assert.Contains(t, out.String(), "deploy/k8s")
}

func TestRunPrintsCapabilities(t *testing.T) {
	var out bytes.Buffer
	require.NoError(t, run([]string{"-capabilities", "minimal"}, &out))
	got := strings.Fields(out.String())
	assert.Contains(t, got, "user")
	assert.Contains(t, got, "access")
	assert.Contains(t, got, "auth")
	assert.NotContains(t, got, "tenant")
}

func TestRunRejectsUnknownProfile(t *testing.T) {
	var out bytes.Buffer
	err := run([]string{"ghost"}, &out)
	require.ErrorContains(t, err, `unknown profile "ghost"`)
}

func TestRunRejectsBadUsage(t *testing.T) {
	var out bytes.Buffer
	require.Error(t, run(nil, &out))
	require.Error(t, run([]string{"full", "extra"}, &out))
}
