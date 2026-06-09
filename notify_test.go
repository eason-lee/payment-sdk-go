package payment

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestVerifyNotify(t *testing.T) {
	body := []byte(`{"notify_type":"payin","order_id":123}`)
	signature := signNotify("secret", "1700000000", "nonce-abc", body)
	if !VerifyNotify(VerifyNotifyRequest{
		Secret:    "secret",
		Timestamp: "1700000000",
		Nonce:     "nonce-abc",
		Signature: signature,
		Body:      body,
	}) {
		t.Fatal("expected valid signature")
	}
}

func TestVerifyNotifyFailsWhenBodyChanges(t *testing.T) {
	body := []byte(`{"notify_type":"payin","order_id":123}`)
	signature := signNotify("secret", "1700000000", "nonce-abc", body)
	if VerifyNotify(VerifyNotifyRequest{
		Secret:    "secret",
		Timestamp: "1700000000",
		Nonce:     "nonce-abc",
		Signature: signature,
		Body:      []byte(`{"notify_type":"payin","order_id":124}`),
	}) {
		t.Fatal("expected invalid signature")
	}
}

func TestVerifyNotifyHTTP(t *testing.T) {
	body := []byte(`{"notify_type":"payin","order_id":123}`)
	header := http.Header{}
	header.Set(headerTimestamp, "1700000000")
	header.Set(headerNonce, "nonce-abc")
	header.Set(headerSignature, signNotify("secret", "1700000000", "nonce-abc", body))
	if !VerifyNotifyHTTP("secret", header, body) {
		t.Fatal("expected valid signature")
	}
}

func TestParseNotify(t *testing.T) {
	event, err := ParseNotify([]byte(`{"notify_type":"payin","order_id":123,"merchant_order_id":"M1"}`))
	if err != nil {
		t.Fatal(err)
	}
	if event.NotifyType != NotifyTypePayin {
		t.Fatalf("notify type = %s", event.NotifyType)
	}

	var payload PayinNotify
	if err := event.Decode(&payload); err != nil {
		t.Fatal(err)
	}
	if payload.OrderID != 123 || payload.MerchantOrderID != "M1" {
		t.Fatalf("unexpected payload: %+v", payload)
	}
}

func TestWriteNotifyAck(t *testing.T) {
	w := httptest.NewRecorder()
	WriteNotifyAck(w)
	if w.Code != http.StatusOK {
		t.Fatalf("status = %d", w.Code)
	}
	if got := w.Body.String(); got != `{"status":"success"}` {
		t.Fatalf("body = %s", got)
	}
}
