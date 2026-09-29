package notification

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestSMSAliyunSend(t *testing.T) {
	var gotPhone, gotSign, gotCode, gotParam string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = r.ParseForm()
		gotPhone = r.FormValue("PhoneNumbers")
		gotSign = r.FormValue("SignName")
		gotCode = r.FormValue("TemplateCode")
		gotParam = r.FormValue("TemplateParam")
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"Code":"OK","Message":"OK","RequestId":"mock-1","BizId":"mock-biz"}`))
	}))
	defer srv.Close()

	sms := NewSMS(SMSConfig{Provider: "aliyun", APIKey: "key", APISecret: "secret", SignName: "jimu-test"})
	sms.setEndpoint(strings.TrimPrefix(srv.URL, "http://"))

	err := sms.Send(context.Background(), Message{
		To:         "13800138000",
		TemplateID: "SMS_123456",
		Data:       map[string]string{"code": "1234"},
	})
	if err != nil {
		t.Fatalf("Send() error = %v", err)
	}
	if gotPhone != "13800138000" || gotSign != "jimu-test" || gotCode != "SMS_123456" {
		t.Errorf("request mismatch: phone=%q sign=%q code=%q", gotPhone, gotSign, gotCode)
	}
	var params map[string]string
	if err := json.Unmarshal([]byte(gotParam), &params); err != nil {
		t.Fatalf("TemplateParam not valid JSON: %v", err)
	}
	if params["code"] != "1234" {
		t.Errorf("template param mismatch: %v", params)
	}
}

func TestSMSAliyunReject(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = r.ParseForm()
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"Code":"isv.MOBILE_NUMBER_ILLEGAL","Message":"号码格式错误","RequestId":"mock-2","BizId":""}`))
	}))
	defer srv.Close()

	sms := NewSMS(SMSConfig{Provider: "aliyun", APIKey: "key", APISecret: "secret", SignName: "s"})
	sms.setEndpoint(strings.TrimPrefix(srv.URL, "http://"))

	err := sms.Send(context.Background(), Message{To: "bad", TemplateID: "SMS_1"})
	if err == nil {
		t.Fatal("expected error for rejected code, got nil")
	}
	if !strings.Contains(err.Error(), "isv.MOBILE_NUMBER_ILLEGAL") {
		t.Errorf("error should surface aliyun code, got: %v", err)
	}
}

func TestSMSUnknownProvider(t *testing.T) {
	sms := NewSMS(SMSConfig{Provider: "nope"})
	if err := sms.Send(context.Background(), Message{}); err == nil {
		t.Fatal("expected error for unknown provider")
	}
}
