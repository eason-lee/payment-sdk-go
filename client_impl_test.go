package payment

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"io"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"
)

var testMerchant = Merchant{ID: 1001, Secret: "merchant-secret"}

func TestCreatePayinSignsRequestLikePaymentGateway(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, err := io.ReadAll(r.Body)
		if err != nil {
			t.Fatal(err)
		}
		want := signGatewayRequest(testMerchant.Secret, r.Method, r.URL.Path, r.Header.Get(headerTimestamp), r.Header.Get(headerNonce), body)
		if got := r.Header.Get(headerSignature); got != want {
			t.Fatalf("signature mismatch\nwant: %s\n got: %s", want, got)
		}
		if got := r.Header.Get(headerTimestamp); got == "" || strings.Contains(got, "T") {
			t.Fatalf("timestamp must be unix seconds, got %q", got)
		}
		_, _ = w.Write([]byte(`{"message":"success","data":{"order_id":2001,"link":"https://pay.example/2001"}}`))
	}))
	defer server.Close()

	transport := http.DefaultTransport.(*http.Transport).Clone()
	transport.Proxy = nil
	client := &ClientImpl{
		baseURL:    server.URL,
		httpClient: &http.Client{Transport: transport},
	}
	resp, err := client.CreatePayin(context.Background(), CreatePayinReq{
		Merchant:        testMerchant,
		MerchantOrderID: "M-2001",
		Amount:          100,
		Currency:        CurrencyTpUSD,
		PayMethod:       PayMethodPayPal,
		PayMode:         PayModePayPalAgreement,
		User:            &User{ID: "u_1001", AppName: "DemoApp"},
		PayPal:          &PayPal{Email: "buyer@example.com"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if resp.OrderID != 2001 {
		t.Fatalf("order id = %d", resp.OrderID)
	}
}

func TestParseNotifyVerifiesAndDecodesPayinPayload(t *testing.T) {
	body := []byte(`{"orderId":2001,"merchantOrderId":"M-2001","status":1,"fee":100}`)
	header := signedNotifyHeader(testMerchant, MerchantNotifyTpPayIn, body)

	resp, err := NewClient(EnvProduction).ParseNotify(context.Background(), &Notify{
		Merchant: testMerchant,
		Header:   header,
		Body:     body,
	})
	if err != nil {
		t.Fatal(err)
	}
	if resp.Tp != MerchantNotifyTpPayIn {
		t.Fatalf("notify type = %d", resp.Tp)
	}
	if resp.PayinOrder == nil || resp.PayinOrder.OrderID != 2001 || resp.PayinOrder.MerchantOrderId != "M-2001" {
		t.Fatalf("payload = %+v", resp.PayinOrder)
	}
}

func TestParseNotifyRejectsBadSignature(t *testing.T) {
	body := []byte(`{"orderId":2001,"merchantOrderId":"M-2001","status":1,"fee":100}`)
	header := signedNotifyHeader(testMerchant, MerchantNotifyTpPayIn, body)
	header.Set(headerSignature, "bad")

	_, err := NewClient(EnvProduction).ParseNotify(context.Background(), &Notify{
		Merchant: testMerchant,
		Header:   header,
		Body:     body,
	})
	if err == nil {
		t.Fatal("expected signature error")
	}
}

func signGatewayRequest(secret, method, path, timestamp, nonce string, body []byte) string {
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte(method))
	mac.Write([]byte("\n"))
	mac.Write([]byte(path))
	mac.Write([]byte("\n"))
	mac.Write([]byte(timestamp))
	mac.Write([]byte("\n"))
	mac.Write([]byte(nonce))
	mac.Write([]byte("\n"))
	mac.Write(body)
	return hex.EncodeToString(mac.Sum(nil))
}

func signedNotifyHeader(merchant Merchant, tp MerchantNotifyTp, body []byte) http.Header {
	header := http.Header{}
	header.Set(headerMerchantID, "1001")
	header.Set(headerTimestamp, "1736214000")
	header.Set(headerNonce, "nonce-abc")
	header.Set(headerNotifyTp, strconv.Itoa(int(tp)))
	mac := hmac.New(sha256.New, []byte(merchant.Secret))
	mac.Write([]byte(header.Get(headerTimestamp)))
	mac.Write([]byte("\n"))
	mac.Write([]byte(header.Get(headerNonce)))
	mac.Write([]byte("\n"))
	mac.Write(body)
	header.Set(headerSignature, hex.EncodeToString(mac.Sum(nil)))
	return header
}
