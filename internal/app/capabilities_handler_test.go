package app

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"jimu/internal/contract"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCapabilitiesHandlerReportsEnabledAndDegraded(t *testing.T) {
	caps := []contract.Descriptor{
		{Name: "user", Requires: []string{}, SoftRequires: []string{"access"}},
		{Name: "auth", SoftRequires: []string{"captcha"}},
	}
	rec := httptest.NewRecorder()
	capabilitiesHandler(caps)(rec, httptest.NewRequest(http.MethodGet, "/capabilities", nil))
	require.Equal(t, http.StatusOK, rec.Code)

	var body struct {
		Enabled  []string `json:"enabled"`
		Degraded []struct {
			Capability string   `json:"capability"`
			Missing    []string `json:"missing"`
		} `json:"degraded"`
	}
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &body))
	assert.Equal(t, []string{"user", "auth"}, body.Enabled)
	require.Len(t, body.Degraded, 2)
	assert.Equal(t, "auth", body.Degraded[1].Capability)
	assert.Equal(t, []string{"captcha"}, body.Degraded[1].Missing)
}
