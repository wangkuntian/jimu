package notification

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"jimu/internal/platform/httpclient"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestWebhookSendSuccess(t *testing.T) {
	var gotBody string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		buf := make([]byte, 1024)
		n, _ := r.Body.Read(buf)
		gotBody = string(buf[:n])
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	w := NewWebhook(WebhookConfig{Headers: map[string]string{"X-Custom": "v1"}}, httpclient.New(httpclient.Config{}))
	err := w.Send(context.Background(), Message{To: srv.URL, Subject: "subj", Body: "body"})
	require.NoError(t, err)
	assert.Contains(t, gotBody, `"subject":"subj"`)
	assert.Equal(t, ChannelWebhook, w.Channel())
}

func TestWebhookSendNon2xx(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
	}))
	defer srv.Close()

	w := NewWebhook(WebhookConfig{}, httpclient.New(httpclient.Config{}))
	err := w.Send(context.Background(), Message{To: srv.URL})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "webhook returned 400")
}

func TestWebhookSendNilClient(t *testing.T) {
	w := NewWebhook(WebhookConfig{}, nil)
	err := w.Send(context.Background(), Message{To: "http://example.com"})
	require.Error(t, err)
}

func TestWebhookSignsPayload(t *testing.T) {
	var gotTS, gotSig string
	var gotBody []byte
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotTS = r.Header.Get("X-Jimu-Timestamp")
		gotSig = r.Header.Get("X-Jimu-Signature")
		gotBody, _ = io.ReadAll(r.Body)
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	w := NewWebhook(WebhookConfig{SignSecret: "secret123"}, httpclient.New(httpclient.Config{}))
	err := w.Send(context.Background(), Message{To: srv.URL, Body: "body"})
	require.NoError(t, err)

	assert.NotEmpty(t, gotTS)
	assert.NotEmpty(t, gotSig)
	// 服务端用同一密钥 + 收到的原始 body 重算，必须与请求头一致
	assert.Equal(t, signPayload("secret123", gotTS, gotBody), gotSig)
	// 篡改 body 后签名不匹配（防篡改验证）
	assert.NotEqual(t, signPayload("secret123", gotTS, append(gotBody, 'x')), gotSig)
	// 换密钥不匹配
	assert.NotEqual(t, signPayload("wrong-secret", gotTS, gotBody), gotSig)
}

func TestWebhookNoSignatureWhenNoSecret(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("X-Jimu-Signature") != "" || r.Header.Get("X-Jimu-Timestamp") != "" {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	w := NewWebhook(WebhookConfig{}, httpclient.New(httpclient.Config{}))
	err := w.Send(context.Background(), Message{To: srv.URL, Body: "body"})
	require.NoError(t, err)
}

func TestWebhookSendBatch(t *testing.T) {
	var calls int
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		calls++
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	w := NewWebhook(WebhookConfig{}, httpclient.New(httpclient.Config{}))
	err := w.SendBatch(context.Background(), []Message{
		{To: srv.URL},
		{To: srv.URL},
	})
	require.NoError(t, err)
	assert.Equal(t, 2, calls)
}
