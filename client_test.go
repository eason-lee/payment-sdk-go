package payment

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func newTestClient(t *testing.T, h http.HandlerFunc) *Client {
	t.Helper()
	server := httptest.NewServer(h)
	t.Cleanup(server.Close)

	client, err := NewClientWithError(Config{
		BaseURL:    server.URL,
		MerchantID: "1001",
		Secret:     "secret",
		Now: func() time.Time {
			return time.Unix(1700000000, 0)
		},
		Nonce: func() (string, error) {
			return "nonce-abc", nil
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	return client
}

func assertSignedRequest(t *testing.T, r *http.Request, method string, path string, body string) {
	t.Helper()
	if r.Method != method {
		t.Fatalf("method = %s", r.Method)
	}
	if r.URL.Path != path {
		t.Fatalf("path = %s", r.URL.Path)
	}
	if got := r.Header.Get(headerMerchantID); got != "1001" {
		t.Fatalf("merchant id = %s", got)
	}
	if got := r.Header.Get(headerTimestamp); got != "1700000000" {
		t.Fatalf("timestamp = %s", got)
	}
	if got := r.Header.Get(headerNonce); got != "nonce-abc" {
		t.Fatalf("nonce = %s", got)
	}
	wantSig := SignRequest("secret", SignRequestInput{
		Method:    method,
		Path:      path,
		Timestamp: "1700000000",
		Nonce:     "nonce-abc",
		Body:      []byte(body),
	})
	if got := r.Header.Get(headerSignature); got != wantSig {
		t.Fatalf("signature mismatch\nwant: %s\n got: %s", wantSig, got)
	}
}

func TestCreatePayin(t *testing.T) {
	client := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		raw := readBody(t, r)
		assertSignedRequest(t, r, http.MethodPost, "/api/payin/order", raw)
		if !strings.Contains(raw, `"merchant_order_id":"PAYIN-1"`) {
			t.Fatalf("body = %s", raw)
		}
		writeJSON(t, w, http.StatusOK, `{"message":"success","data":{"order_id":123,"link":"https://pay.example/123"}}`)
	})

	got, err := client.CreatePayin(context.Background(), CreatePayinRequest{
		MerchantOrderID: "PAYIN-1",
		Amount:          10000,
		Currency:        "USD",
		PayMethod:       PayMethodPayPal,
		PayMode:         PayModePayPalAgreement,
		User:            &PayinUser{UserID: "u1", AppName: "app"},
		PayPal:          &PayPal{Email: "buyer@example.com"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if got.OrderID != 123 || got.Link == "" {
		t.Fatalf("unexpected response: %+v", got)
	}
}

func TestGetPayin(t *testing.T) {
	client := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		assertSignedRequest(t, r, http.MethodGet, "/api/payin/order/123", "")
		writeJSON(t, w, http.StatusOK, `{"message":"success","data":{"order_id":123,"merchant_order_id":"PAYIN-1","status":"SUCCESS","amount":10000,"currency":"USD","method":"PAYPAL"}}`)
	})

	got, err := client.GetPayin(context.Background(), 123)
	if err != nil {
		t.Fatal(err)
	}
	if got.Status != "SUCCESS" || got.MerchantOrderID != "PAYIN-1" {
		t.Fatalf("unexpected response: %+v", got)
	}
}

func TestRefundPayin(t *testing.T) {
	client := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		raw := readBody(t, r)
		assertSignedRequest(t, r, http.MethodPost, "/api/payin/order/123/refund", raw)
		writeJSON(t, w, http.StatusOK, `{"message":"success","data":null}`)
	})

	if err := client.RefundPayin(context.Background(), 123, RefundPayinRequest{Amount: 100}); err != nil {
		t.Fatal(err)
	}
}

func TestCreatePayout(t *testing.T) {
	client := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		raw := readBody(t, r)
		assertSignedRequest(t, r, http.MethodPost, "/api/payout/order", raw)
		writeJSON(t, w, http.StatusOK, `{"message":"success","data":{"order_id":456,"channel_order_id":"ch_1"}}`)
	})

	got, err := client.CreatePayout(context.Background(), CreatePayoutRequest{
		MerchantOrderID: "PAYOUT-1",
		Amount:          5000,
		Currency:        "PHP",
		PayMethod:       PayMethodGCash,
		Account:         "09171234567",
	})
	if err != nil {
		t.Fatal(err)
	}
	if got.OrderID != 456 || got.ChannelOrderID != "ch_1" {
		t.Fatalf("unexpected response: %+v", got)
	}
}

func TestGetPayout(t *testing.T) {
	client := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		assertSignedRequest(t, r, http.MethodGet, "/api/payout/order/456", "")
		writeJSON(t, w, http.StatusOK, `{"message":"success","data":{"order_id":456,"merchant_order_id":"PAYOUT-1","status":"SUCCESS","amount":5000,"currency":"PHP"}}`)
	})

	got, err := client.GetPayout(context.Background(), 456)
	if err != nil {
		t.Fatal(err)
	}
	if got.Status != "SUCCESS" || got.MerchantOrderID != "PAYOUT-1" {
		t.Fatalf("unexpected response: %+v", got)
	}
}

func TestAPIError(t *testing.T) {
	client := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		writeJSON(t, w, http.StatusBadRequest, `{"message":"签名错误"}`)
	})

	_, err := client.GetPayin(context.Background(), 123)
	var apiErr *APIError
	if !errors.As(err, &apiErr) {
		t.Fatalf("expected APIError, got %T %v", err, err)
	}
	if apiErr.StatusCode != http.StatusBadRequest || apiErr.Message != "签名错误" {
		t.Fatalf("unexpected APIError: %+v", apiErr)
	}
}

func readBody(t *testing.T, r *http.Request) string {
	t.Helper()
	raw, err := io.ReadAll(r.Body)
	if err != nil {
		t.Fatal(err)
	}
	return string(raw)
}

func writeJSON(t *testing.T, w http.ResponseWriter, status int, body string) {
	t.Helper()
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_, err := w.Write([]byte(body))
	if err != nil {
		t.Fatal(err)
	}
}
