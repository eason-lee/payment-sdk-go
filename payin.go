package payment

import (
	"context"
	"fmt"
	"net/http"
)

// CreatePayinRequest creates a pay-in order.
type CreatePayinRequest struct {
	MerchantOrderID string      `json:"merchant_order_id"`
	Amount          int64       `json:"amount"`
	Currency        string      `json:"currency"`
	PayMethod       string      `json:"pay_method"`
	PayMode         string      `json:"pay_mode"`
	User            *PayinUser  `json:"user,omitempty"`
	PayPal          *PayPal     `json:"paypal,omitempty"`
	ApplePay        *ApplePay   `json:"apple_pay,omitempty"`
	CreditCard      *CreditCard `json:"credit_card,omitempty"`
	GooglePay       *GooglePay  `json:"google_pay,omitempty"`
}

type PayinUser struct {
	UserID    string `json:"user_id"`
	AppName   string `json:"app_name"`
	UserName  string `json:"user_name,omitempty"`
	UserEmail string `json:"user_email,omitempty"`
	UserPhone string `json:"user_phone,omitempty"`
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

type CreatePayinResponse struct {
	OrderID int64  `json:"order_id"`
	Link    string `json:"link"`
}

type PayinOrder struct {
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

type RefundPayinRequest struct {
	Amount float64 `json:"amount"`
}

func (c *Client) CreatePayin(ctx context.Context, req CreatePayinRequest) (*CreatePayinResponse, error) {
	var out CreatePayinResponse
	if err := c.do(ctx, http.MethodPost, "/api/payin/order", req, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

func (c *Client) GetPayin(ctx context.Context, orderID int64) (*PayinOrder, error) {
	var out PayinOrder
	if err := c.do(ctx, http.MethodGet, fmt.Sprintf("/api/payin/order/%d", orderID), nil, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

func (c *Client) RefundPayin(ctx context.Context, orderID int64, req RefundPayinRequest) error {
	return c.do(ctx, http.MethodPost, fmt.Sprintf("/api/payin/order/%d/refund", orderID), req, nil)
}
