package main

import (
	"context"
	"errors"
	"io"
	"log"
	"net/http"

	payment "github.com/eason-lee/payment-sdk-go"
)

func main() {
	http.HandleFunc("/payment/notify", notifyHandler)
	log.Fatal(http.ListenAndServe(":8080", nil))
}

func notifyHandler(w http.ResponseWriter, r *http.Request) {
	body, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "read body failed", http.StatusBadRequest)
		return
	}
	client := payment.NewClient(payment.Production())
	notify := &payment.Notify{
		Header: r.Header,
		Body:   body,
	}
	identity, err := client.GetNotifyIdentity(notify)
	if err != nil {
		http.Error(w, "invalid payload", http.StatusBadRequest)
		return
	}
	secret, err := loadMerchantSecret(r.Context(), identity.MerchantID, identity.OrderID)
	if err != nil {
		http.Error(w, "invalid merchant", http.StatusBadRequest)
		return
	}
	event, err := client.ParseNotify(r.Context(), secret, notify)
	if err != nil {
		http.Error(w, "invalid payload", http.StatusBadRequest)
		return
	}

	switch event.Tp {
	case payment.MerchantNotifyTpPayIn:
		// Handle idempotently by event.PayinOrder.OrderID or MerchantOrderId.
	case payment.MerchantNotifyTpPayOut:
		// Handle idempotently by event.PayoutOrder.OrderID or MerchantOrderId.
	case payment.MerchantNotifyTpPayInRefund:
		// Handle refund by event.RefundOrder.RefundOrderId.
	case payment.MerchantNotifyTpPayInDispute:
		// Handle dispute by event.DisputeOrder.DisputeOrderId.
	case payment.MerchantNotifyTpPayInAgreement:
		// Handle agreement by event.AgreementOrder.AgreementID.
	case payment.MerchantNotifyTpPayInCardBind:
		// Handle card bind by event.CardBind.SourceID.
	case payment.MerchantNotifyTpPayInFraud:
		// Handle fraud by event.FraudOrder.OrderID.
	}

	client.NotifySuccess(w)
}

func loadMerchantSecret(ctx context.Context, merchantID string, orderID string) (string, error) {
	_ = ctx
	_ = orderID
	if merchantID != "1001" {
		return "", errors.New("merchant not found")
	}
	return "test-secret", nil
}
