package payment

const (
	NotifyTypePayin                = "payin"
	NotifyTypePayinRefund          = "payin_refund"
	NotifyTypePayinDispute         = "payin_dispute"
	NotifyTypePayout               = "payout"
	NotifyTypePayinAgreement       = "payin_agreement"
	NotifyTypePayinCancelAgreement = "payin_cancel_agreement"
)

type PayinNotify struct {
	NotifyType        string `json:"notify_type"`
	OrderID           int64  `json:"order_id"`
	MerchantOrderID   string `json:"merchant_order_id"`
	PayMethod         string `json:"pay_method,omitempty"`
	Currency          string `json:"currency,omitempty"`
	AmountCents       int64  `json:"amount_cents,omitempty"`
	ActualAmountCents int64  `json:"actual_amount_cents,omitempty"`
	Settle            string `json:"settle,omitempty"`
	FailReason        string `json:"fail_reason,omitempty"`
}

type PayoutNotify struct {
	NotifyType        string `json:"notify_type"`
	OrderID           int64  `json:"order_id"`
	MerchantOrderID   string `json:"merchant_order_id"`
	Currency          string `json:"currency,omitempty"`
	AmountCents       int64  `json:"amount_cents,omitempty"`
	ActualAmountCents int64  `json:"actual_amount_cents,omitempty"`
	FailReason        string `json:"fail_reason,omitempty"`
}

type PayinRefundNotify struct {
	NotifyType      string `json:"notify_type"`
	OrderID         int64  `json:"order_id"`
	RefundOrderID   int64  `json:"refund_order_id"`
	MerchantOrderID string `json:"merchant_order_id"`
	Status          string `json:"status"`
	AmountCents     int64  `json:"amount_cents"`
	Currency        string `json:"currency"`
	FailReason      string `json:"fail_reason,omitempty"`
}

type PayinAgreementNotify struct {
	NotifyType  string `json:"notify_type"`
	MerchantID  int64  `json:"merchantId"`
	UserID      string `json:"userId"`
	AgreementID string `json:"agreementId"`
	PayerEmail  string `json:"payerEmail"`
}
