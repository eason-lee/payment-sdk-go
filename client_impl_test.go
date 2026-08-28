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
		_, _ = w.Write([]byte(`{"message":"success","data":{"order_id":"2001","channel_order_id":"PP-ORDER-2001","link":"https://pay.example/2001"}}`))
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
		Address:         &Address{Country: "US", Address: "100 Main St", State: "NY", City: "New York", Zip: "10001"},
		PayPal:          &PayPal{Email: "buyer@example.com"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if resp.OrderID != "2001" {
		t.Fatalf("order id = %s", resp.OrderID)
	}
	if resp.ChannelOrderID != "PP-ORDER-2001" {
		t.Fatalf("channel order id = %s", resp.ChannelOrderID)
	}
}

func TestCreatePayinCreditFlowRequestAndResponse(t *testing.T) {
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
		if _, ok := payload["country"]; ok {
			t.Fatalf("request must not send top-level country: %s", body)
		}
		if _, ok := payload["checkout"]; ok {
			t.Fatalf("request must not send checkout: %s", body)
		}
		addr, ok := payload["address"].(map[string]any)
		if !ok {
			t.Fatalf("address missing: %s", body)
		}
		if addr["country"] != "US" || addr["address"] != "100 Main St" || addr["zip"] != "10001" {
			t.Fatalf("address = %#v", addr)
		}
		if addr["state"] != "NY" || addr["city"] != "New York" {
			t.Fatalf("address = %#v", addr)
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
		PayMethod:       PayMethodCreditCard,
		PayMode:         PayModeCreditFlow,
		User:            &User{ID: "u_1001", AppName: "DemoApp"},
		Address:         &Address{Country: "US", Address: "100 Main St", State: "NY", City: "New York", Zip: "10001"},
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

func TestCreatePayinCreditSourceIdRequestUsesSourceIDField(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, err := io.ReadAll(r.Body)
		if err != nil {
			t.Fatal(err)
		}
		var payload map[string]any
		if err := json.Unmarshal(body, &payload); err != nil {
			t.Fatal(err)
		}
		card, ok := payload["credit_card"].(map[string]any)
		if !ok {
			t.Fatalf("credit_card missing: %s", body)
		}
		if card["source_id"] != "src_2003" {
			t.Fatalf("credit_card.source_id = %v", card["source_id"])
		}
		if _, ok := card["token"]; ok {
			t.Fatalf("request must not send token: %s", body)
		}
		if card["card_name"] != "John Doe" || card["last4"] != "4242" {
			t.Fatalf("credit_card = %#v", card)
		}
		_, _ = w.Write([]byte(`{"message":"success","data":{"order_id":"2003"}}`))
	}))
	defer server.Close()

	client := testClient(server.URL)
	resp, err := client.CreatePayin(context.Background(), &CreatePayinReq{
		Merchant:        testMerchant,
		MerchantOrderID: "M-2003",
		Amount:          1000,
		Currency:        CurrencyTpUSD,
		PayMethod:       PayMethodCreditCard,
		PayMode:         PayModeCreditSourceID,
		Address:         &Address{Country: "US"},
		CreditCard: &CreditCard{
			SourceID:    "src_2003",
			CardName:    "John Doe",
			Last4:       "4242",
			ExpiryMonth: "12",
			ExpiryYear:  "2030",
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	if resp.OrderID != "2003" {
		t.Fatalf("order id = %s", resp.OrderID)
	}
}

func TestSubmitCreditFlowSendsCardRiskInfoFields(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/payin/order/s/2002/checkout/submit" {
			t.Fatalf("path = %s", r.URL.Path)
		}
		body, err := io.ReadAll(r.Body)
		if err != nil {
			t.Fatal(err)
		}
		var payload map[string]any
		if err := json.Unmarshal(body, &payload); err != nil {
			t.Fatal(err)
		}
		card, ok := payload["card"].(map[string]any)
		if !ok {
			t.Fatalf("card missing: %s", body)
		}
		if card["card_name"] != "John Doe" || card["last4"] != "4242" || card["country"] != "US" {
			t.Fatalf("card = %#v", card)
		}
		if _, ok := card["name_on_card"]; ok {
			t.Fatalf("request must not send name_on_card: %s", body)
		}
		_, _ = w.Write([]byte(`{"message":"success","data":{"order_id":"2002","status":"pending"}}`))
	}))
	defer server.Close()

	client := testClient(server.URL)
	resp, err := client.SubmitCreditFlow(context.Background(), &SubmitCreditFlowReq{
		Merchant:    testMerchant,
		OrderID:     "2002",
		SessionID:   "ps_2002",
		SessionData: "session-data",
		Card: &CardRiskSnapshot{
			CardName:    "John Doe",
			Last4:       "4242",
			ExpiryMonth: "12",
			ExpiryYear:  "2030",
			Country:     "US",
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	if resp.OrderID != "2002" {
		t.Fatalf("order id = %s", resp.OrderID)
	}
}

func TestGetPayinEscapesStringOrderIDInPath(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.EscapedPath() != "/api/payin/order/s/order%2F2001" {
			t.Fatalf("escaped path = %s", r.URL.EscapedPath())
		}
		_, _ = w.Write([]byte(`{"message":"success","data":{"order_id":"order/2001","merchant_order_id":"M-2001","status":1,"amount":100,"currency":"USD","pay_method":"PayPal"}}`))
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
		_, _ = w.Write([]byte(`{"message":"success","data":{"order_id":"2001","merchant_order_id":"M-2001","status":1,"amount":100,"has_refund":true,"currency":"USD","pay_method":"PayPal","refund":{"refund_order_id":"8001","status":2,"out_refund_id":"ch-rf-1","amount":100}}}`))
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
	if !resp.HasRefund || resp.Refund == nil || resp.Refund.RefundOrderID != "8001" || resp.Refund.Amount != 100 {
		t.Fatalf("refund = %#v has_refund=%v", resp.Refund, resp.HasRefund)
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

func TestParseNotifyRefundFailedStatus(t *testing.T) {
	body := []byte(`{"orderId":"2001","merchantOrderId":"M-2001","refundOrderId":"3001","status":3,"failReason":"declined"}`)
	resp, err := NewClient(EnvProduction).ParseNotify(context.Background(), testMerchant.Secret, &Notify{
		Header: signedNotifyHeader(testMerchant, MerchantNotifyTpPayInRefund, body),
		Body:   body,
	})
	if err != nil {
		t.Fatal(err)
	}
	if resp.Tp != MerchantNotifyTpPayInRefund {
		t.Fatalf("notify type = %d", resp.Tp)
	}
	if resp.RefundOrder == nil || resp.RefundOrder.Status != RefundStatusFailed || resp.RefundOrder.RefundOrderId != "3001" {
		t.Fatalf("payload = %+v", resp.RefundOrder)
	}
}

func TestParseNotifyDisputeCanceledStatus(t *testing.T) {
	body := []byte(`{"orderId":"2001","merchantOrderId":"M-2001","disputeOrderId":"4001","amount":100,"status":4}`)
	resp, err := NewClient(EnvProduction).ParseNotify(context.Background(), testMerchant.Secret, &Notify{
		Header: signedNotifyHeader(testMerchant, MerchantNotifyTpPayInDispute, body),
		Body:   body,
	})
	if err != nil {
		t.Fatal(err)
	}
	if resp.DisputeOrder == nil || resp.DisputeOrder.Status != DisputeStatusCanceled || resp.DisputeOrder.DisputeOrderId != "4001" {
		t.Fatalf("payload = %+v", resp.DisputeOrder)
	}
}

func TestParseNotifyCardBind(t *testing.T) {
	body := []byte(`{"orderId":"2001","merchantOrderId":"M-2001","userId":"u1","sourceId":"src_1","last4":"4242","bin":"424242"}`)
	resp, err := NewClient(EnvProduction).ParseNotify(context.Background(), testMerchant.Secret, &Notify{
		Header: signedNotifyHeader(testMerchant, MerchantNotifyTpPayInCardBind, body),
		Body:   body,
	})
	if err != nil {
		t.Fatal(err)
	}
	if resp.Tp != MerchantNotifyTpPayInCardBind {
		t.Fatalf("notify type = %d", resp.Tp)
	}
	if resp.CardBind == nil || resp.CardBind.SourceID != "src_1" || resp.CardBind.Last4 != "4242" || resp.CardBind.UserID != "u1" {
		t.Fatalf("payload = %+v", resp.CardBind)
	}
}

func TestParseNotifyFraud(t *testing.T) {
	body := []byte(`{"orderId":"2001","merchantOrderId":"M-2001","userId":"u1","reason":"stolen","type":"fraud","eci":"05"}`)
	resp, err := NewClient(EnvProduction).ParseNotify(context.Background(), testMerchant.Secret, &Notify{
		Header: signedNotifyHeader(testMerchant, MerchantNotifyTpPayInFraud, body),
		Body:   body,
	})
	if err != nil {
		t.Fatal(err)
	}
	if resp.Tp != MerchantNotifyTpPayInFraud {
		t.Fatalf("notify type = %d", resp.Tp)
	}
	if resp.FraudOrder == nil || resp.FraudOrder.Reason != "stolen" || resp.FraudOrder.Eci != "05" {
		t.Fatalf("payload = %+v", resp.FraudOrder)
	}
}

func TestParseNotifyRejectsUnknownType(t *testing.T) {
	body := []byte(`{"orderId":"2001"}`)
	_, err := NewClient(EnvProduction).ParseNotify(context.Background(), testMerchant.Secret, &Notify{
		Header: signedNotifyHeader(testMerchant, MerchantNotifyTp(99), body),
		Body:   body,
	})
	if err == nil {
		t.Fatal("expected unknown notify type error")
	}
}

func TestAppealDisputeSignsAndOmitsSecret(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/payin/order/s/9001/dispute/appeal" {
			t.Fatalf("path = %s", r.URL.Path)
		}
		body, err := io.ReadAll(r.Body)
		if err != nil {
			t.Fatal(err)
		}
		want := signGatewayRequest(testMerchant.Secret, r.Method, r.URL.Path, r.Header.Get(headerTimestamp), r.Header.Get(headerNonce), body)
		if got := r.Header.Get(headerSignature); got != want {
			t.Fatalf("signature mismatch\nwant: %s\n got: %s", want, got)
		}
		if strings.Contains(string(body), "merchant-secret") {
			t.Fatalf("request body leaked merchant secret: %s", body)
		}
		var payload map[string]any
		if err := json.Unmarshal(body, &payload); err != nil {
			t.Fatal(err)
		}
		if payload["file_url"] != "https://files.example/a.pdf" {
			t.Fatalf("file_url = %v", payload["file_url"])
		}
		if payload["notes"] != "delivered" {
			t.Fatalf("notes = %v", payload["notes"])
		}
		_, _ = w.Write([]byte(`{"message":"success"}`))
	}))
	defer server.Close()

	err := testClient(server.URL).AppealDispute(context.Background(), &AppealDisputeReq{
		Merchant: testMerchant,
		OrderID:  "9001",
		FileURL:  "https://files.example/a.pdf",
		Notes:    "delivered",
	})
	if err != nil {
		t.Fatal(err)
	}
}

func TestAppealDisputeRequiresFileURL(t *testing.T) {
	err := testClient("http://127.0.0.1").AppealDispute(context.Background(), &AppealDisputeReq{
		Merchant: testMerchant,
		OrderID:  "9001",
	})
	if err == nil {
		t.Fatal("expected validation error")
	}

}

func TestBindPayPalAgreementSignsAndOmitsSecret(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/payin/order/agreement" {
			t.Fatalf("path = %s", r.URL.Path)
		}
		body, err := io.ReadAll(r.Body)
		if err != nil {
			t.Fatal(err)
		}
		want := signGatewayRequest(testMerchant.Secret, r.Method, r.URL.Path, r.Header.Get(headerTimestamp), r.Header.Get(headerNonce), body)
		if got := r.Header.Get(headerSignature); got != want {
			t.Fatalf("signature mismatch\nwant: %s\n got: %s", want, got)
		}
		if strings.Contains(string(body), "merchant-secret") {
			t.Fatalf("request body leaked merchant secret: %s", body)
		}
		var payload map[string]any
		if err := json.Unmarshal(body, &payload); err != nil {
			t.Fatal(err)
		}
		if payload["currency"] != string(CurrencyTpUSD) {
			t.Fatalf("currency = %v", payload["currency"])
		}
		if payload["app_name"] != "DemoApp" {
			t.Fatalf("app_name = %v", payload["app_name"])
		}
		if payload["user_id"] != "u_1001" {
			t.Fatalf("user_id = %v", payload["user_id"])
		}
		_, _ = w.Write([]byte(`{"message":"success","data":{"token":"tok-1","link":"https://paypal.example/agree","redirect_link":"https://merchant.example/return","status":"PENDING"}}`))
	}))
	defer server.Close()

	resp, err := testClient(server.URL).BindPayPalAgreement(context.Background(), &BindPayPalAgreementReq{
		Merchant: testMerchant,
		Currency: CurrencyTpUSD,
		AppName:  "DemoApp",
		UserID:   "u_1001",
	})
	if err != nil {
		t.Fatal(err)
	}
	if resp.Token != "tok-1" || resp.Link == "" || resp.Status != "PENDING" {
		t.Fatalf("resp = %+v", resp)
	}
}

func TestCancelPayPalAgreementMapsUserIDToPlayerID(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/payin/order/agreement/cancel" {
			t.Fatalf("path = %s", r.URL.Path)
		}
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
		if payload["paypalEmail"] != "buyer@example.com" {
			t.Fatalf("paypalEmail = %v", payload["paypalEmail"])
		}
		if payload["app_name"] != "DemoApp" {
			t.Fatalf("app_name = %v", payload["app_name"])
		}
		if payload["player_id"] != "u_1001" {
			t.Fatalf("player_id = %v", payload["player_id"])
		}
		if _, ok := payload["user_id"]; ok {
			t.Fatalf("cancel body must not send user_id: %s", body)
		}
		_, _ = w.Write([]byte(`{"message":"success"}`))
	}))
	defer server.Close()

	err := testClient(server.URL).CancelPayPalAgreement(context.Background(), &CancelPayPalAgreementReq{
		Merchant:    testMerchant,
		AppName:     "DemoApp",
		UserID:      "u_1001",
		PayPalEmail: "buyer@example.com",
	})
	if err != nil {
		t.Fatal(err)
	}
}

func TestApprovePayPalAgreementSendsToken(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/payin/order/agreement/approve" {
			t.Fatalf("path = %s", r.URL.Path)
		}
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
		if payload["token"] != "tok-1" {
			t.Fatalf("token = %v", payload["token"])
		}
		_, _ = w.Write([]byte(`{"message":"success"}`))
	}))
	defer server.Close()

	err := testClient(server.URL).ApprovePayPalAgreement(context.Background(), &ApprovePayPalAgreementReq{
		Merchant: testMerchant,
		Token:    "tok-1",
	})
	if err != nil {
		t.Fatal(err)
	}
}

func TestBindPayPalAgreementRequiresUserID(t *testing.T) {
	_, err := testClient("http://127.0.0.1").BindPayPalAgreement(context.Background(), &BindPayPalAgreementReq{
		Merchant: testMerchant,
		Currency: CurrencyTpUSD,
		AppName:  "DemoApp",
	})
	if err == nil {
		t.Fatal("expected validation error")
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
