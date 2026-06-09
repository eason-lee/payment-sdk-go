# Payment Go SDK

Go SDK for payment merchant gateway.

## Install

```bash
go get github.com/bigword/payment-sdk-go
```

## Create Client

```go
client := payment.NewClient(payment.Config{
	BaseURL:    "https://api.example.com",
	MerchantID: "1001",
	Secret:    "merchant-secret",
})
```

`Secret` must stay on merchant server. Never expose it in browser, mobile app, or client-side bundle.

## Create Payin

```go
resp, err := client.CreatePayin(ctx, payment.CreatePayinRequest{
	MerchantOrderID: "PAYIN-202606090001",
	Amount:          10000,
	Currency:        "USD",
	PayMethod:       payment.PayMethodPayPal,
	PayMode:         payment.PayModePayPalAgreement,
	User: &payment.PayinUser{
		UserID:  "u_1001",
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
resp, err := client.CreatePayout(ctx, payment.CreatePayoutRequest{
	MerchantOrderID: "PAYOUT-202606090001",
	Amount:          50000,
	Currency:        "PHP",
	PayMethod:       payment.PayMethodGCash,
	Account:         "09171234567",
	User: &payment.PayoutUser{
		ID:      "u_1001",
		AppName: "DemoApp",
		Name:    "John Doe",
		Email:   "john@example.com",
	},
})
```

## Query Orders

```go
payin, err := client.GetPayin(ctx, 10201312003)
payout, err := client.GetPayout(ctx, 10201312010)
```

## Refund Payin

```go
err := client.RefundPayin(ctx, 10201312003, payment.RefundPayinRequest{
	Amount: 10000,
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

	if !payment.VerifyNotifyHTTP("merchant-secret", r.Header, body) {
		http.Error(w, "invalid signature", http.StatusUnauthorized)
		return
	}

	event, err := payment.ParseNotify(body)
	if err != nil {
		http.Error(w, "invalid payload", http.StatusBadRequest)
		return
	}

	switch event.NotifyType {
	case payment.NotifyTypePayin:
		var payload payment.PayinNotify
		if err := event.Decode(&payload); err != nil {
			http.Error(w, "invalid payin payload", http.StatusBadRequest)
			return
		}
		// Handle idempotently by payload.OrderID or payload.MerchantOrderID.
	}

	payment.WriteNotifyAck(w)
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
