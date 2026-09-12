// internal/platform/oauth/provider_test.go
package oauth

import (
	"context"
	"testing"

	"jimu/internal/platform/httpclient"

	"github.com/stretchr/testify/assert"
)

func TestProvidersImplementInterface(t *testing.T) {
	var _ Provider = (*GoogleProvider)(nil)
	var _ Provider = (*GitHubProvider)(nil)
	var _ Provider = (*WeChatProvider)(nil)
}

func TestGoogleAuthURL(t *testing.T) {
	p := NewGoogleProvider(GoogleConfig{
		ClientID:     "id",
		ClientSecret: "secret",
		RedirectURL:  "http://localhost:8080/api/v1/oauth/google/callback",
	}, httpclient.New(httpclient.Config{}))
	url, err := p.AuthURL(context.Background(), "state123")
	assert.NoError(t, err)
	assert.Contains(t, url, "state=state123")
	assert.Contains(t, url, "client_id=id")
}
