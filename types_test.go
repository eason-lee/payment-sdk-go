package payment

import "testing"

func TestPayModeValidMatchesPayMethod(t *testing.T) {
	if err := PayModeCreditFlow.Valid(PayMethodCreditCard); err != nil {
		t.Fatalf("CreditFlow + CreditCard: %v", err)
	}
	if err := PayModeCreditFlow.Valid(PayMethodPayPal); err == nil {
		t.Fatal("CreditFlow + PayPal should fail")
	}
	if err := PayModePayPalAgreement.Valid(PayMethodPayPal); err != nil {
		t.Fatalf("PayPalAgreement + PayPal: %v", err)
	}
	if err := PayModeCashAppWeb.Valid(PayMethodCashApp); err != nil {
		t.Fatalf("CashAppWeb + CashApp: %v", err)
	}
	if err := PayModeGooglePayNative.Valid(PayMethodGooglePay); err != nil {
		t.Fatalf("GooglePayNative + GooglePay: %v", err)
	}
	if err := PayModeGooglePayWeb.Valid(PayMethodGooglePay); err != nil {
		t.Fatalf("GooglePayWeb + GooglePay: %v", err)
	}
}

func TestCreatePayinReqValidCreditFlowAllowsMissingCityState(t *testing.T) {
	req := CreatePayinReq{
		Merchant:        testMerchant,
		MerchantOrderID: "M-1",
		Amount:          100,
		Currency:        CurrencyTpUSD,
		PayMethod:       PayMethodCreditCard,
		PayMode:         PayModeCreditFlow,
		Address:         &Address{Country: "US"},
	}
	if err := req.Valid(); err != nil {
		t.Fatalf("Valid() = %v", err)
	}
}

func TestCreatePayinReqValidCreditSourceIdRequiresSourceID(t *testing.T) {
	req := CreatePayinReq{
		Merchant:        testMerchant,
		MerchantOrderID: "M-1",
		Amount:          100,
		Currency:        CurrencyTpUSD,
		PayMethod:       PayMethodCreditCard,
		PayMode:         PayModeCreditSourceID,
		Address:         &Address{Country: "US"},
	}
	if err := req.Valid(); err == nil {
		t.Fatal("missing credit_card.source_id should fail")
	}
	req.CreditCard = &CreditCard{SourceID: "src_1"}
	if err := req.Valid(); err != nil {
		t.Fatalf("Valid() = %v", err)
	}
	req.PayMode = "CreditToken"
	if err := req.Valid(); err == nil {
		t.Fatal("CreditToken should fail")
	}
}

func TestCreatePayinReqValidGooglePayNativeDoesNotRequireToken(t *testing.T) {
	req := CreatePayinReq{
		Merchant:        testMerchant,
		MerchantOrderID: "M-1",
		Amount:          100,
		Currency:        CurrencyTpUSD,
		PayMethod:       PayMethodGooglePay,
		PayMode:         PayModeGooglePayNative,
		Address:         &Address{Country: "US"},
	}
	if err := req.Valid(); err != nil {
		t.Fatalf("GooglePayNative Create Valid() = %v", err)
	}
	req.PayMode = PayModeGooglePayWeb
	if err := req.Valid(); err != nil {
		t.Fatalf("GooglePayWeb Valid() = %v", err)
	}
}

func TestSubmitReqValidRequiresCredentials(t *testing.T) {
	req := SubmitReq{Merchant: testMerchant, OrderID: "1"}
	if err := req.Valid(); err == nil {
		t.Fatal("empty submit should fail")
	}
	req.SessionID = "ps_x"
	req.SessionData = "data"
	if err := req.Valid(); err == nil {
		t.Fatal("CreditFlow without card should fail")
	}
	req.Card = &CardRiskSnapshot{Last4: "4242"}
	if err := req.Valid(); err != nil {
		t.Fatalf("CreditFlow Valid() = %v", err)
	}
	native := SubmitReq{
		Merchant:  testMerchant,
		OrderID:   "2",
		GooglePay: &GooglePay{Signature: "sig", SignedMessage: "msg", ProtocolVersion: "ECv2"},
	}
	if err := native.Valid(); err != nil {
		t.Fatalf("GooglePayNative Submit Valid() = %v", err)
	}
}

func TestPayinOrderStatusAlignsGateway(t *testing.T) {
	if PayinOrderStatusWaitingSubmit != 1 {
		t.Fatalf("WaitingSubmit = %d, want 1", PayinOrderStatusWaitingSubmit)
	}
	if PayinOrderStatusProcessing != 2 {
		t.Fatalf("Processing = %d, want 2", PayinOrderStatusProcessing)
	}
	if PayinOrderStatusCanceled != 3 {
		t.Fatalf("Canceled = %d, want 3", PayinOrderStatusCanceled)
	}
	if PayinOrderStatusSuccess != 4 {
		t.Fatalf("Success = %d, want 4", PayinOrderStatusSuccess)
	}
}

func TestNotifyContractValues(t *testing.T) {
	if MerchantNotifyTpPayInCardBind != 6 {
		t.Fatalf("MerchantNotifyTpPayInCardBind = %d, want 6", MerchantNotifyTpPayInCardBind)
	}
	if MerchantNotifyTpPayInFraud != 7 {
		t.Fatalf("MerchantNotifyTpPayInFraud = %d, want 7", MerchantNotifyTpPayInFraud)
	}
	if RefundStatusFailed != 3 {
		t.Fatalf("RefundStatusFailed = %d, want 3", RefundStatusFailed)
	}
	if DisputeStatusCanceled != 4 {
		t.Fatalf("DisputeStatusCanceled = %d, want 4", DisputeStatusCanceled)
	}
}
