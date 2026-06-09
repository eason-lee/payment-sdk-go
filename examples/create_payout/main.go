package main

import (
	"context"
	"fmt"
	"log"
	"os"

	payment "github.com/bigword/payment-sdk-go"
)

func main() {
	client := payment.NewClient(payment.Config{
		BaseURL:    os.Getenv("PAYMENT_BASE_URL"),
		MerchantID: os.Getenv("PAYMENT_MERCHANT_ID"),
		Secret:     os.Getenv("PAYMENT_SECRET"),
	})

	resp, err := client.CreatePayout(context.Background(), payment.CreatePayoutRequest{
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
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("order_id=%d channel_order_id=%s\n", resp.OrderID, resp.ChannelOrderID)
}
