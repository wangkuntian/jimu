package contract

import (
	"context"
	"testing"
)

func TestWithClientInfoSetsIPAndUserAgent(t *testing.T) {
	ctx := WithClientInfo(context.Background(), "1.2.3.4", "ua-one")
	info := ClientInfoFrom(ctx)
	if info.IP != "1.2.3.4" || info.UserAgent != "ua-one" {
		t.Fatalf("info = %+v, want ip=1.2.3.4 ua=ua-one", info)
	}
}

func TestWithClientInfoOverridePreservesOtherFields(t *testing.T) {
	ctx := WithLoginDevice(context.Background(), "dev-token", true)
	ctx = WithClientInfo(ctx, "5.6.7.8", "new")
	info := ClientInfoFrom(ctx)
	if info.IP != "5.6.7.8" || info.UserAgent != "new" {
		t.Fatalf("overridden info = %+v", info)
	}
	if info.DeviceToken != "dev-token" || !info.RememberDevice {
		t.Fatalf("must preserve device fields: %+v", info)
	}
}

func TestClientInfoFromEmptyContext(t *testing.T) {
	if info := ClientInfoFrom(context.Background()); info.IP != "" || info.UserAgent != "" || info.DeviceToken != "" || info.RememberDevice {
		t.Fatalf("empty ctx info = %+v", info)
	}
}
