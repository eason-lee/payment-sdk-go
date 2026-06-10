package payment

import (
	"context"
	"net/http"
	"time"
)

// Client calls payment merchant gateway APIs.
type Client interface {
	CreatePayin(ctx context.Context, req CreatePayinReq) (*CreatePayinResp, error)
	GetPayin(ctx context.Context, req GetPayinReq) (*PayinOrderResp, error)
	RefundPayin(ctx context.Context, req RefundPayinReq) error
	CreatePayout(ctx context.Context, req CreatePayoutReq) (*CreatePayoutResp, error)
	GetPayout(ctx context.Context, req GetPayoutReq) (*PayoutOrderResp, error)
	NotifyClient
}

type GetPayinReq struct {
	Merchant
	OrderID int64 `json:"order_id" validate:"required"`
}

func (r GetPayinReq) Valid() error {
	return ValidStruct(r)
}

type GetPayoutReq struct {
	Merchant
	OrderID int64 `json:"order_id" validate:"required"`
}

func (r GetPayoutReq) Valid() error {
	return ValidStruct(r)
}

type NotifyClient interface {
	ParseNotify(ctx context.Context, notify *Notify) (*NotifyResp, error)
	NotifySuccess(w http.ResponseWriter)
	NotifyFailed(w http.ResponseWriter)
}

func NewClient(env EnvType) *ClientImpl {
	transport := http.DefaultTransport.(*http.Transport).Clone()
	transport.Proxy = nil

	return &ClientImpl{
		baseURL: env.GetBaseURL(),
		httpClient: &http.Client{
			Transport: transport,
			Timeout:   15 * time.Second,
		},
	}
}

type Merchant struct {
	ID     int64  `json:"id" validate:"gt=0"`
	Secret string `json:"secret" validate:"required"`
}

type CreatePayinReq struct {
	Merchant
	MerchantOrderID string      `json:"merchant_order_id" validate:"required"`
	Amount          int64       `json:"amount" validate:"gt=0"`
	Currency        CurrencyTp  `json:"currency" validate:"oneof=USD"`
	PayMethod       PayMethod   `json:"pay_method" validate:"oneof=PayPal"`
	PayMode         PayMode     `json:"pay_mode" validate:"required"`
	User            *User       `json:"user,omitempty"`
	PayPal          *PayPal     `json:"paypal,omitempty"`
	ApplePay        *ApplePay   `json:"apple_pay,omitempty"`
	CreditCard      *CreditCard `json:"credit_card,omitempty"`
	GooglePay       *GooglePay  `json:"google_pay,omitempty"`
}

func (r CreatePayinReq) Valid() error {
	if err := r.PayMode.Valid(); err != nil {
		return err
	}
	return ValidStruct(r)
}

type User struct {
	ID      string `json:"id" validate:"required"`
	AppName string `json:"app_name" validate:"required"`
	Name    string `json:"name,omitempty"`
	Email   string `json:"email,omitempty"`
	Phone   string `json:"phone,omitempty"`
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
	Merchant
	OrderID int64   `json:"order_id" validate:"required"`
	Amount  float64 `json:"amount" validate:"gt=0"`
}

func (r RefundPayinReq) Valid() error {
	return ValidStruct(r)
}

type CreatePayoutReq struct {
	Merchant
	MerchantOrderID string     `json:"merchant_order_id" validate:"required"`
	Amount          int64      `json:"amount" validate:"gt=0"`
	Currency        CurrencyTp `json:"currency" validate:"oneof=USD"`
	PayMethod       PayMethod  `json:"pay_method" validate:"oneof=PayPal"`
	Account         string     `json:"account" validate:"required"`
	User            *User      `json:"user,omitempty"`
}

func (r CreatePayoutReq) Valid() error {
	return ValidStruct(r)
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
