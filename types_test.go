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

func TestPayinOrderStatusCanceledValue(t *testing.T) {
	if PayinOrderStatusCanceled != 6 {
		t.Fatalf("PayinOrderStatusCanceled = %d, want 6", PayinOrderStatusCanceled)
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
