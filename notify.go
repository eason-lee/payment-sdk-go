package payment

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"strings"
)

type NotifyResp struct {
	MerchantID     int64                     `json:"merchant_id"`
	Tp             MerchantNotifyTp          `json:"tp"`
	PayinOrder     *NotifyOrderPayload       `json:"payin_order,omitempty"`
	RefundOrder    *NotifyRefundOrderPayload `json:"refund_order,omitempty"`
	DisputeOrder   *NotifyDisputePayload     `json:"dispute_order,omitempty"`
	PayoutOrder    *NotifyOrderPayload       `json:"payout_order,omitempty"`
	AgreementOrder *NotifyAgreementPayload   `json:"agreement_order,omitempty"`
}

type NotifyReq struct {
	Merchant  Merchant
	Tp        MerchantNotifyTp
	Timestamp string
	Nonce     string
	Signature string
	Body      []byte
}

func (n *NotifyReq) Verify() bool {
	mac := hmac.New(sha256.New, []byte(n.Merchant.Secret))
	mac.Write([]byte(n.Timestamp))
	mac.Write([]byte("\n"))
	mac.Write([]byte(n.Nonce))
	mac.Write([]byte("\n"))
	mac.Write(n.Body)
	expected := hex.EncodeToString(mac.Sum(nil))
	return hmac.Equal([]byte(strings.TrimSpace(n.Signature)), []byte(expected))
}

func (n *NotifyReq) ToNotifyResp() (*NotifyResp, error) {

	out := &NotifyResp{
		MerchantID: n.Merchant.ID,
		Tp:         n.Tp,
	}

	switch n.Tp {
	case MerchantNotifyTpPayIn:
		var payload NotifyOrderPayload
		if err := json.Unmarshal(n.Body, &payload); err != nil {
			return nil, err
		}
		out.PayinOrder = &payload
	case MerchantNotifyTpPayOut:
		out.Tp = MerchantNotifyTpPayOut
		var payload NotifyOrderPayload
		if err := json.Unmarshal(n.Body, &payload); err != nil {
			return nil, err
		}
		out.PayoutOrder = &payload
	case MerchantNotifyTpPayInRefund:
		out.Tp = MerchantNotifyTpPayInRefund
		var payload NotifyRefundOrderPayload
		if err := json.Unmarshal(n.Body, &payload); err != nil {
			return nil, err
		}
		out.RefundOrder = &payload
	case MerchantNotifyTpPayInDispute:
		out.Tp = MerchantNotifyTpPayInDispute
		var payload NotifyDisputePayload
		if err := json.Unmarshal(n.Body, &payload); err != nil {
			return nil, err
		}
		out.DisputeOrder = &payload
	case MerchantNotifyTpPayInAgreement:
		out.Tp = MerchantNotifyTpPayInAgreement
		var payload NotifyAgreementPayload
		if err := json.Unmarshal(n.Body, &payload); err != nil {
			return nil, err
		}
		out.AgreementOrder = &payload
	default:
		return nil, errors.New("payment: unknown notify type")
	}
	return out, nil
}

type Notify struct {
	Merchant
	Header http.Header
	Body   []byte
}

func (n *Notify) Valid() error {
	mid, err := strconv.ParseInt(strings.TrimSpace(n.Header.Get(headerMerchantID)), 10, 64)
	if err != nil {
		return errors.New("payment: invalid notify merchant id")
	}
	if mid != n.Merchant.ID {
		return errors.New("payment: merchant id in header not equal to merchant id")
	}

	return nil
}

func (n *Notify) ToNotifyReq() (*NotifyReq, error) {
	notifyTp, err := strconv.Atoi(n.Header.Get(headerNotifyTp))
	if err != nil {
		return nil, err
	}
	return &NotifyReq{
		Merchant:  n.Merchant,
		Tp:        MerchantNotifyTp(notifyTp),
		Timestamp: n.Header.Get(headerTimestamp),
		Nonce:     n.Header.Get(headerNonce),
		Signature: n.Header.Get(headerSignature),
		Body:      n.Body,
	}, nil
}

type NotifyOrderStatus uint8

const (
	NotifyOrderStatusSuccess NotifyOrderStatus = iota + 1
	NotifyOrderStatusFailed
)

type NotifyOrderPayload struct {
	OrderID         int64             `json:"orderId"`
	MerchantOrderId string            `json:"merchantOrderId"`
	Status          NotifyOrderStatus `json:"status"`
	Fee             Money             `json:"fee"`
	FailReason      string            `json:"failReason,omitempty"`
}

// RefundStatus 退款状态。
type RefundStatus uint8

const (
	RefundStatusPending  RefundStatus = 1 // 待处理
	RefundStatusSuccess  RefundStatus = 2 // 成功
	RefundStatusDeclined RefundStatus = 3 // 拒绝
	RefundStatusFailed   RefundStatus = 4 // 失败
)

type NotifyRefundOrderPayload struct {
	OrderID         int64        `json:"orderId"`
	MerchantOrderId string       `json:"merchantOrderId"`
	RefundOrderId   int64        `json:"refundOrderId"`
	Status          RefundStatus `json:"status"`
	FailReason      string       `json:"failReason,omitempty"`
}

// DisputeStatus 表示 payin 领域内的争议状态。
type DisputeStatus uint8

const (
	DisputeStatusReceived DisputeStatus = 1 // 收到争议
	DisputeStatusWin      DisputeStatus = 2 // 争议赢了
	DisputeStatusLose     DisputeStatus = 3 // 争议输了
)

type NotifyDisputePayload struct {
	OrderID         int64         `json:"orderId"`
	MerchantOrderId string        `json:"merchantOrderId"`
	DisputeOrderId  int64         `json:"disputeOrderId"`
	Amount          Money         `json:"amount"`
	Status          DisputeStatus `json:"status"`
}

type NotifyAgreementStatus uint8

const (
	NotifyAgreementStatusApproved NotifyAgreementStatus = iota + 1
	NotifyAgreementStatusCancelled
)

type NotifyAgreementPayload struct {
	OrderID         int64                 `json:"orderId"`
	MerchantOrderId string                `json:"merchantOrderId"`
	AgreementID     int64                 `json:"agreementId"`
	Email           string                `json:"email"`
	UserID          string                `json:"userId"`
	Status          NotifyAgreementStatus `json:"status"`
}
