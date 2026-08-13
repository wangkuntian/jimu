package notification

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"jimu/internal/platform/httpclient"
)

func BenchmarkWebhookSend(b *testing.B) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	w := NewWebhook(WebhookConfig{}, httpclient.New(httpclient.Config{}))
	msg := Message{To: srv.URL, Subject: "bench", Body: "payload"}
	ctx := context.Background()

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if err := w.Send(ctx, msg); err != nil {
			b.Fatal(err)
		}
	}
}
