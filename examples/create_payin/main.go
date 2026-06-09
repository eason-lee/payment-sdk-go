package main

import (
	"context"
	"fmt"
	"log"
	"os"

	payment "github.com/bigword/payment-sdk-go"
)

func main() {
	client := payment.NewClient(payment.Production())
	merchant := payment.Merchant{
		ID:     os.Getenv("PAYMENT_MERCHANT_ID"),
		Secret: os.Getenv("PAYMENT_SECRET"),
	}

	resp, err := client.CreatePayin(context.Background(), merchant, payment.CreatePayinReq{
		MerchantOrderID: "PAYIN-202606090001",
		Amount:          10000,
		Currency:        "USD",
		PayMethod:       payment.PayMethodPayPal,
		PayMode:         payment.PayModePayPalAgreement,
		User: &payment.PayinUser{
			UserID:  "u_1001",
			AppName: "DemoApp",
		},
		PayPal: &payment.PayPal{Email: "buyer@example.com"},
	})
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("order_id=%d link=%s\n", resp.OrderID, resp.Link)
}
