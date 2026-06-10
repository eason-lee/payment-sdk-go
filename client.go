package payment

import (
	"context"
	"net/http"
)

// Client calls payment merchant gateway APIs.
type Client interface {
	CreatePayin(ctx context.Context, req CreatePayinReq) (*CreatePayinResp, error)
	GetPayin(ctx context.Context, merchant Merchant, orderID int64) (*PayinOrderResp, error)
	RefundPayin(ctx context.Context, merchant Merchant, orderID int64, req RefundPayinReq) error
	CreatePayout(ctx context.Context, req CreatePayoutReq) (*CreatePayoutResp, error)
	GetPayout(ctx context.Context, merchant Merchant, orderID int64) (*PayoutOrderResp, error)
	NotifyClient
}

type NotifyClient interface {
	HandleNotify(ctx context.Context, notify *Notify) (*NotifyResp, error)
	NotifySuccess(w http.ResponseWriter)
	NotifyFailed(w http.ResponseWriter)
}

func NewClient(env EnvType) *ClientImpl {
	return &ClientImpl{
		baseURL: env.GetBaseURL(),
	}
}

type Merchant struct {
	ID     int64  `validate:"required"`
	Secret string `validate:"required"`
}

type CreatePayinReq struct {
	Merchant
	MerchantOrderID string      `json:"merchant_order_id" validate:"required"`
	Amount          int64       `json:"amount" validate:"gt=0"`
	Currency        CurrencyTp  `json:"currency" validate:"oneof=USD"`
	PayMethod       PayMethod   `json:"pay_method" validate:"oneof=PayPal"`
	PayMode         PayMode     `json:"pay_mode" validate:"validateFn=Valid"`
	User            *User       `json:"user,omitempty"`
	PayPal          *PayPal     `json:"paypal,omitempty"`
	ApplePay        *ApplePay   `json:"apple_pay,omitempty"`
	CreditCard      *CreditCard `json:"credit_card,omitempty"`
	GooglePay       *GooglePay  `json:"google_pay,omitempty"`
}

type User struct {
	ID      string `json:"user_id"`
	AppName string `json:"app_name"`
	Name    string `json:"user_name,omitempty"`
	Email   string `json:"user_email,omitempty"`
	Phone   string `json:"user_phone,omitempty"`
}

type PayPal struct {
	Email string `json:"email"`
}

type ApplePay struct {
	AppleToken string `json:"apple_token,omitempty"`
	Country    string `json:"country,omitempty"`
	City       string `json:"city,omitempty"`
	State      string `json:"state,omitempty"`
}

type CreditCard struct {
	SourceID    string `json:"source_id,omitempty"`
	CreditToken string `json:"credit_token,omitempty"`
	Address     string `json:"address,omitempty"`
	ZipCode     string `json:"zip_code,omitempty"`
	Country     string `json:"country,omitempty"`
	City        string `json:"city,omitempty"`
	State       string `json:"state,omitempty"`
}

type GooglePay struct {
	Signature       string `json:"signature,omitempty"`
	SignedMessage   string `json:"signed_message,omitempty"`
	ProtocolVersion string `json:"protocol_version,omitempty"`
}

type CreatePayinResp struct {
	OrderID int64  `json:"order_id"`
	Link    string `json:"link"`
}

type PayinOrderResp struct {
	OrderID         int64  `json:"order_id"`
	MerchantOrderID string `json:"merchant_order_id"`
	Status          string `json:"status"`
	Channel         string `json:"channel,omitempty"`
	ChannelOrderID  string `json:"channel_order_id,omitempty"`
	FailReason      string `json:"fail_reason,omitempty"`
	Amount          int64  `json:"amount"`
	RefundedAmount  int64  `json:"refunded_amount,omitempty"`
	Currency        string `json:"currency"`
	Method          string `json:"method"`
}

type RefundPayinReq struct {
	Amount float64 `json:"amount"`
}

type CreatePayoutReq struct {
	Merchant
	MerchantOrderID string     `json:"merchant_order_id"`
	Amount          int64      `json:"amount"`
	Currency        CurrencyTp `json:"currency"`
	PayMethod       PayMethod  `json:"pay_method"`
	Account         string     `json:"account"`
	User            *User      `json:"user,omitempty"`
}

type CreatePayoutResp struct {
	OrderID        int64  `json:"order_id"`
	ChannelOrderID string `json:"channel_order_id"`
}

type PayoutOrderResp struct {
	OrderID         int64  `json:"order_id"`
	MerchantOrderID string `json:"merchant_order_id"`
	Status          string `json:"status"`
	PayMethod       string `json:"pay_method,omitempty"`
	ReferenceNo     string `json:"reference_no,omitempty"`
	InvoiceNo       string `json:"invoice_no,omitempty"`
	FailReason      string `json:"fail_reason,omitempty"`
	Amount          int64  `json:"amount"`
	Currency        string `json:"currency"`
	AppName         string `json:"app_name,omitempty"`
}
