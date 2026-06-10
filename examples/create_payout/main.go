package main

import (
	"context"
	"fmt"
	"log"

	payment "github.com/bigword/payment-sdk-go"
)

func main() {
	client := payment.NewClient(payment.Production())
	merchant := payment.Merchant{
		ID:     1001,
		Secret: "xxxx",
	}

	resp, err := client.CreatePayout(context.Background(), payment.CreatePayoutReq{
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
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("order_id=%d channel_order_id=%s\n", resp.OrderID, resp.ChannelOrderID)
}
