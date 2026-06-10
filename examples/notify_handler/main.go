package main

import (
	"io"
	"log"
	"net/http"
	"os"

	payment "github.com/bigword/payment-sdk-go"
)

func main() {
	merchant := payment.Merchant{
		ID:     1001,
		Secret: os.Getenv("PAYMENT_SECRET"),
	}
	http.HandleFunc("/payment/notify", func(w http.ResponseWriter, r *http.Request) {
		notifyHandler(w, r, merchant)
	})
	log.Fatal(http.ListenAndServe(":8080", nil))
}

func notifyHandler(w http.ResponseWriter, r *http.Request, merchant payment.Merchant) {
	body, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "read body failed", http.StatusBadRequest)
		return
	}
	client := payment.NewClient(payment.Production())
	event, err := client.ParseNotify(
		r.Context(),
		&payment.Notify{
			Merchant: merchant,
			Header:   r.Header,
			Body:     body,
		},
	)
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
