package main

import (
	"context"
	"fmt"
	"log"

	payment "github.com/eason-lee/payment-sdk-go"
)

func main() {
	client := payment.NewClient(payment.Production())
	merchant := payment.Merchant{
		ID:     "1001",
		Secret: "xxxx",
	}

	resp, err := client.CreatePayin(context.Background(), payment.CreatePayinReq{
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
		PayPal: &payment.PayPal{Email: "buyer@example.com"},
	})
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("order_id=%s link=%s\n", resp.OrderID, resp.Link)
}
