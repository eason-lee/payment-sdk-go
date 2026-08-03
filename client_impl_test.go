package payment

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"
	"time"
)

var testMerchant = Merchant{ID: "1001", Secret: "merchant-secret"}

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
		if strings.Contains(string(body), "merchant-secret") {
			t.Fatalf("request body leaked merchant secret: %s", body)
		}
		_, _ = w.Write([]byte(`{"message":"success","data":{"order_id":"2001","link":"https://pay.example/2001"}}`))
	}))
	defer server.Close()

	client := testClient(server.URL)
	resp, err := client.CreatePayin(context.Background(), &CreatePayinReq{
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
	if resp.OrderID != "2001" {
		t.Fatalf("order id = %s", resp.OrderID)
	}
}

func TestCreatePayinCheckoutCreditFlowRequestAndResponse(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, err := io.ReadAll(r.Body)
		if err != nil {
			t.Fatal(err)
		}
		want := signGatewayRequest(testMerchant.Secret, r.Method, r.URL.Path, r.Header.Get(headerTimestamp), r.Header.Get(headerNonce), body)
		if got := r.Header.Get(headerSignature); got != want {
			t.Fatalf("signature mismatch\nwant: %s\n got: %s", want, got)
		}

		var payload map[string]any
		if err := json.Unmarshal(body, &payload); err != nil {
			t.Fatal(err)
		}
		if payload["pay_method"] != string(PayMethodCreditCard) {
			t.Fatalf("pay_method = %v", payload["pay_method"])
		}
		if payload["pay_mode"] != string(PayModeCreditFlow) {
			t.Fatalf("pay_mode = %v", payload["pay_mode"])
		}
		if payload["country"] != "US" {
			t.Fatalf("country = %v", payload["country"])
		}
		checkout := payload["checkout"].(map[string]any)
		if checkout["address"] != "100 Main St" || checkout["zip_code"] != "10001" {
			t.Fatalf("checkout = %#v", checkout)
		}
		if checkout["state"] != "NY" || checkout["city"] != "New York" {
			t.Fatalf("checkout = %#v", checkout)
		}
		if _, ok := payload["credit_card"]; ok {
			t.Fatalf("request should use unified checkout, got credit_card in %s", body)
		}

		_, _ = w.Write([]byte(`{"message":"success","data":{"order_id":"2002","payment_session":{"id":"ps_2002","token":"pst_2002","secret":"pss_2002"}}}`))
	}))
	defer server.Close()

	client := testClient(server.URL)
	resp, err := client.CreatePayin(context.Background(), &CreatePayinReq{
		Merchant:        testMerchant,
		MerchantOrderID: "M-2002",
		Amount:          1000,
		Currency:        CurrencyTpUSD,
		Country:         "US",
		PayMethod:       PayMethodCreditCard,
		PayMode:         PayModeCreditFlow,
		User:            &User{ID: "u_1001", AppName: "DemoApp"},
		Checkout: &Checkout{
			Address: "100 Main St",
			ZipCode: "10001",
			State:   "NY",
			City:    "New York",
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	if resp.OrderID != "2002" {
		t.Fatalf("order id = %s", resp.OrderID)
	}
	if resp.PaymentSession == nil {
		t.Fatal("payment session is nil")
	}
	if resp.PaymentSession.ID != "ps_2002" || resp.PaymentSession.Token != "pst_2002" || resp.PaymentSession.Secret != "pss_2002" {
		t.Fatalf("payment session = %#v", resp.PaymentSession)
	}
}

func TestGetPayinEscapesStringOrderIDInPath(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.EscapedPath() != "/api/payin/order/s/order%2F2001" {
			t.Fatalf("escaped path = %s", r.URL.EscapedPath())
		}
		_, _ = w.Write([]byte(`{"message":"success","data":{"order_id":"order/2001","merchant_order_id":"M-2001","status":1,"amount":100,"refunded_amount":0,"currency":"USD","pay_method":"PayPal"}}`))
	}))
	defer server.Close()

	client := testClient(server.URL)
	_, err := client.GetPayin(context.Background(), &GetPayinReq{
		Merchant: testMerchant,
		OrderID:  "order/2001",
	})
	if err != nil {
		t.Fatal(err)
	}
}

func TestGetPayinDecodesMerchantOrderStatus(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/payin/order/s/2001" {
			t.Fatalf("path = %s", r.URL.Path)
		}
		_, _ = w.Write([]byte(`{"message":"success","data":{"order_id":"2001","merchant_order_id":"M-2001","status":1,"amount":100,"refunded_amount":0,"currency":"USD","pay_method":"PayPal"}}`))
	}))
	defer server.Close()

	client := testClient(server.URL)
	resp, err := client.GetPayin(context.Background(), &GetPayinReq{
		Merchant: testMerchant,
		OrderID:  "2001",
	})
	if err != nil {
		t.Fatal(err)
	}
	if resp.Status != PayinOrderStatusProcessing {
		t.Fatalf("status = %d", resp.Status)
	}
	if resp.OrderID != "2001" || resp.MerchantOrderID != "M-2001" {
		t.Fatalf("resp = %+v", resp)
	}
	if resp.PayMethod != PayMethodPayPal {
		t.Fatalf("pay method = %s", resp.PayMethod)
	}
	if resp.Currency != CurrencyTpUSD {
		t.Fatalf("currency = %s", resp.Currency)
	}
}

func TestGetPayoutDecodesMerchantOrderStatus(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/payout/order/s/3001" {
			t.Fatalf("path = %s", r.URL.Path)
		}
		_, _ = w.Write([]byte(`{"message":"success","data":{"order_id":"3001","merchant_order_id":"P-3001","status":2,"pay_method":"PayPal","amount":100,"currency":"USD"}}`))
	}))
	defer server.Close()

	client := testClient(server.URL)
	resp, err := client.GetPayout(context.Background(), &GetPayoutReq{
		Merchant: testMerchant,
		OrderID:  "3001",
	})
	if err != nil {
		t.Fatal(err)
	}
	if resp.Status != PayoutOrderStatusSuccess {
		t.Fatalf("status = %d", resp.Status)
	}
	if resp.OrderID != "3001" || resp.MerchantOrderID != "P-3001" {
		t.Fatalf("resp = %+v", resp)
	}
	if resp.Currency != CurrencyTpUSD {
		t.Fatalf("currency = %s", resp.Currency)
	}
}

func TestParseNotifyVerifiesAndDecodesPayinPayload(t *testing.T) {
	body := []byte(`{"orderId":"2001","merchantOrderId":"M-2001","status":1,"fee":100}`)
	header := signedNotifyHeader(testMerchant, MerchantNotifyTpPayIn, body)

	client := NewClient(EnvProduction)
	notify := &Notify{Header: header, Body: body}
	identity, err := client.GetNotifyIdentity(notify)
	if err != nil {
		t.Fatal(err)
	}
	if identity.MerchantID != testMerchant.ID || identity.OrderID != "2001" || identity.Tp != MerchantNotifyTpPayIn {
		t.Fatalf("identity = %+v", identity)
	}

	resp, err := client.ParseNotify(context.Background(), testMerchant.Secret, notify)
	if err != nil {
		t.Fatal(err)
	}
	if resp.Tp != MerchantNotifyTpPayIn {
		t.Fatalf("notify type = %d", resp.Tp)
	}
	if resp.PayinOrder == nil || resp.PayinOrder.OrderID != "2001" || resp.PayinOrder.MerchantOrderId != "M-2001" {
		t.Fatalf("payload = %+v", resp.PayinOrder)
	}
}

func TestParseNotifyRejectsBadSignature(t *testing.T) {
	body := []byte(`{"orderId":"2001","merchantOrderId":"M-2001","status":1,"fee":100}`)
	header := signedNotifyHeader(testMerchant, MerchantNotifyTpPayIn, body)
	header.Set(headerSignature, "bad")

	_, err := NewClient(EnvProduction).ParseNotify(context.Background(), testMerchant.Secret, &Notify{
		Header: header,
		Body:   body,
	})
	if err == nil {
		t.Fatal("expected signature error")
	}
}

func TestParseNotifyRejectsExpiredTimestamp(t *testing.T) {
	body := []byte(`{"orderId":"2001","merchantOrderId":"M-2001","status":1,"fee":100}`)
	header := signedNotifyHeader(testMerchant, MerchantNotifyTpPayIn, body)
	header.Set(headerTimestamp, strconv.FormatInt(time.Now().Add(-10*time.Minute).Unix(), 10))
	header.Set(headerSignature, notifySignature(testMerchant.Secret, header.Get(headerTimestamp), header.Get(headerNonce), header.Get(headerNotifyTp), header.Get(headerOrderID), body))

	_, err := NewClient(EnvProduction).ParseNotify(context.Background(), testMerchant.Secret, &Notify{
		Header: header,
		Body:   body,
	})
	if err == nil {
		t.Fatal("expected expired timestamp error")
	}
}

func TestParseNotifyRejectsMismatchedHeaderOrderID(t *testing.T) {
	body := []byte(`{"orderId":"2001","merchantOrderId":"M-2001","status":1,"fee":100}`)
	header := signedNotifyHeader(testMerchant, MerchantNotifyTpPayIn, body)
	header.Set(headerOrderID, "2002")
	header.Set(headerSignature, notifySignature(testMerchant.Secret, header.Get(headerTimestamp), header.Get(headerNonce), header.Get(headerNotifyTp), header.Get(headerOrderID), body))

	_, err := NewClient(EnvProduction).ParseNotify(context.Background(), testMerchant.Secret, &Notify{
		Header: header,
		Body:   body,
	})
	if err == nil {
		t.Fatal("expected mismatched order id error")
	}
}

func testClient(baseURL string) *ClientImpl {
	transport := http.DefaultTransport.(*http.Transport).Clone()
	transport.Proxy = nil
	return &ClientImpl{
		baseURL:    baseURL,
		httpClient: &http.Client{Transport: transport},
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
	header.Set(headerMerchantID, merchant.ID)
	header.Set(headerTimestamp, strconv.FormatInt(time.Now().Unix(), 10))
	header.Set(headerNonce, "nonce-abc")
	header.Set(headerNotifyTp, strconv.Itoa(int(tp)))
	header.Set(headerOrderID, "2001")
	header.Set(headerSignature, notifySignature(merchant.Secret, header.Get(headerTimestamp), header.Get(headerNonce), header.Get(headerNotifyTp), header.Get(headerOrderID), body))
	return header
}

func notifySignature(secret, timestamp, nonce, notifyTp, orderID string, body []byte) string {
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte(timestamp))
	mac.Write([]byte("\n"))
	mac.Write([]byte(nonce))
	mac.Write([]byte("\n"))
	mac.Write([]byte(notifyTp))
	mac.Write([]byte("\n"))
	mac.Write([]byte(orderID))
	mac.Write([]byte("\n"))
	mac.Write(body)
	return hex.EncodeToString(mac.Sum(nil))
}
