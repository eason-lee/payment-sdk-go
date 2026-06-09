package main

import (
	"io"
	"log"
	"net/http"
	"os"

	payment "github.com/bigword/payment-sdk-go"
)

func main() {
	secret := os.Getenv("PAYMENT_SECRET")
	http.HandleFunc("/payment/notify", func(w http.ResponseWriter, r *http.Request) {
		notifyHandler(w, r, secret)
	})
	log.Fatal(http.ListenAndServe(":8080", nil))
}

func notifyHandler(w http.ResponseWriter, r *http.Request, secret string) {
	body, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "read body failed", http.StatusBadRequest)
		return
	}

	if !payment.VerifyNotifyHTTP(secret, r.Header, body) {
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
	case payment.NotifyTypePayout:
		var payload payment.PayoutNotify
		if err := event.Decode(&payload); err != nil {
			http.Error(w, "invalid payout payload", http.StatusBadRequest)
			return
		}
		// Handle idempotently by payload.OrderID or payload.MerchantOrderID.
	}

	payment.WriteNotifyAck(w)
}
