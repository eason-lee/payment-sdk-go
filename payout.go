package payment

import (
	"context"
	"fmt"
	"net/http"
)

type CreatePayoutRequest struct {
	MerchantOrderID string      `json:"merchant_order_id"`
	Amount          int64       `json:"amount"`
	Currency        string      `json:"currency"`
	PayMethod       string      `json:"pay_method"`
	Account         string      `json:"account"`
	User            *PayoutUser `json:"user,omitempty"`
}

type PayoutUser struct {
	ID      string `json:"id"`
	AppName string `json:"app_name"`
	Name    string `json:"name,omitempty"`
	Email   string `json:"email,omitempty"`
	Phone   string `json:"phone,omitempty"`
}

type CreatePayoutResponse struct {
	OrderID        int64  `json:"order_id"`
	ChannelOrderID string `json:"channel_order_id"`
}

type PayoutOrder struct {
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

func (c *Client) CreatePayout(ctx context.Context, req CreatePayoutRequest) (*CreatePayoutResponse, error) {
	var out CreatePayoutResponse
	if err := c.do(ctx, http.MethodPost, "/api/payout/order", req, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

func (c *Client) GetPayout(ctx context.Context, orderID int64) (*PayoutOrder, error) {
	var out PayoutOrder
	if err := c.do(ctx, http.MethodGet, fmt.Sprintf("/api/payout/order/%d", orderID), nil, &out); err != nil {
		return nil, err
	}
	return &out, nil
}
