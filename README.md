# Payment Go SDK

Go SDK for payment merchant gateway.

## Install

```bash
go get github.com/eason-lee/payment-sdk-go@v0.1.1
```

## Create Client

```go
client := payment.NewClient(payment.Production())
merchant := payment.Merchant{
	ID:     1001,
	Secret: "merchant-secret",
}
```

`Production()` points to `http://192.168.1.171`.
`Sandbox()` is currently used for local verification and points to `http://127.0.0.1:8080`.
When sandbox domain becomes separate, only client init needs to switch:

```go
client := payment.NewClient(payment.Sandbox())
```

`Merchant.Secret` must stay on merchant server. Never expose it in browser, mobile app, or client-side bundle.

## Create Payin

```go
resp, err := client.CreatePayin(ctx, payment.CreatePayinReq{
	Merchant:        merchant,
	MerchantOrderID: "PAYIN-202606090001",
	Amount:          10000,
	Currency:        "USD",
	PayMethod:       payment.PayMethodPayPal,
	PayMode:         payment.PayModePayPalAgreement,
	User: &payment.User{
		ID:      "u_1001",
		AppName: "DemoApp",
	},
	PayPal: &payment.PayPal{
		Email: "buyer@example.com",
	},
})
```

Use `resp.Link` to redirect payer when gateway returns a payment link.

## Create Payout

```go
resp, err := client.CreatePayout(ctx, payment.CreatePayoutReq{
	Merchant:        merchant,
	MerchantOrderID: "PAYOUT-202606090001",
	Amount:          50000,
	Currency:        "USD",
	PayMethod:       payment.PayMethodPayPal,
	Account:         "payer@example.com",
	User: &payment.User{
		ID:      "u_1001",
		AppName: "DemoApp",
		Name:    "John Doe",
		Email:   "john@example.com",
	},
})
```

## Query Orders

```go
payin, err := client.GetPayin(ctx, payment.GetPayinReq{
	Merchant: merchant,
	OrderID:  10201312003,
})
payout, err := client.GetPayout(ctx, payment.GetPayoutReq{
	Merchant: merchant,
	OrderID:  10201312010,
})
```

## Refund Payin

```go
err := client.RefundPayin(ctx, payment.RefundPayinReq{
	Merchant: merchant,
	OrderID:  10201312003,
	Amount:   10000,
})
```

## Receive Notify

Notification verification must use raw request body.

```go
func notifyHandler(w http.ResponseWriter, r *http.Request) {
	body, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "read body failed", http.StatusBadRequest)
		return
	}

	merchant := payment.Merchant{
		ID:     1001,
		Secret: "merchant-secret",
	}
	client := payment.NewClient(payment.Production())
	event, err := client.ParseNotify(r.Context(), &payment.Notify{
		Merchant: merchant,
		Header:   r.Header,
		Body:     body,
	})
	if err != nil {
		http.Error(w, "invalid payload", http.StatusBadRequest)
		return
	}

	switch event.Tp {
	case payment.MerchantNotifyTpPayIn:
		// Handle idempotently by event.PayinOrder.OrderID or MerchantOrderId.
	case payment.MerchantNotifyTpPayOut:
		// Handle idempotently by event.PayoutOrder.OrderID or MerchantOrderId.
	}

	client.NotifySuccess(w)
}
```

## Error Handling

```go
var apiErr *payment.APIError
if errors.As(err, &apiErr) {
	fmt.Println(apiErr.StatusCode, apiErr.Message)
}
```

## Security Notes

- Keep merchant secret on server.
- Do not retry write APIs automatically unless your business idempotency is clear.
- Verify notification signature before parsing business payload.
- Store notification handling result idempotently before returning ack.
