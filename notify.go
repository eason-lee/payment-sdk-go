package payment

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"net/http"
	"strings"
)

type NotifyResp struct {
	MerchantID            int64                    `json:"merchant_id"`
	Tp                    MerchantNotifyTp          `json:"tp"`
	PayinPaylod           *NotifyOrderPaylod        `json:"payin_payload,omitempty"`
	PayoutPaylod          *NotifyOrderPaylod        `json:"payout_payload,omitempty"`
	PayinRefundPaylod     *NotifyAgreementPaylod    `json:"payin_refund_payload,omitempty"`
	PayinDisputePaylod    *NotifyDisputeOrderPaylod `json:"payin_dispute_payload,omitempty"`
	PayoutAgreementPaylod *NotifyAgreementPaylod    `json:"payout_agreement_payload,omitempty"`
}

type NotifyReq struct {
	Merchant  Merchant
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
	var raw map[string]any
	if err := json.Unmarshal(n.Body, &raw); err != nil {
		return nil, err
	}
	notifyType, _ := raw["notify_type"]
	return &NotifyResp{
		MerchantID: n.Merchant.ID,
		Tp:         MerchantNotifyTp(notifyType),
	}, nil
}

type Notify struct {
	Merchant
	header http.Header
	body   []byte
}

func (n *Notify) Validate() error {
	mid := mustParseInt64(n.header.Get(headerMerchantID))
	if mid != n.Merchant.ID {
		return errors.New("payment: merchant id in header not equal to merchant id in body")
	}
	
	return ValidStruct(n)
}

func (n *Notify) ToNotifyReq() *NotifyReq {
	return &NotifyReq{
		Merchant: n.Merchant,
		Timestamp: n.header.Get(headerTimestamp),
		Nonce:     n.header.Get(headerNonce),
		Signature: n.header.Get(headerSignature),
		Body:      n.body,
	}
}

func WriteNotifyAck(w http.ResponseWriter) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte(`{"status":"success"}`))
}

type NotifyOrderStatus uint8

const (
	NotifyOrderStatusSuccess NotifyOrderStatus = iota + 1
	NotifyOrderStatusFailed
)

type NotifyOrderPaylod struct {
	OrderID         int64             `json:"orderId" validate:"required"`         // 订单ID
	MerchantOrderId string            `json:"merchantOrderId" validate:"required"` // 商户订单ID
	Status          NotifyOrderStatus `json:"status" validate:"oneof=1 2"`         // 订单状态
	Fee             Money             `json:"fee" `                                // 手续费
	FailReason      string            `json:"failReason,omitempty"`                // 失败原因
}

// RefundStatus 退款状态。
type RefundStatus uint8

const (
	RefundStatusPending  RefundStatus = 1 // 待处理
	RefundStatusSuccess  RefundStatus = 2 // 成功
	RefundStatusDeclined RefundStatus = 3 // 拒绝
	RefundStatusFailed   RefundStatus = 4 // 失败
)

type NotifyRefundOrderPaylod struct {
	OrderID         int64        `json:"orderId" validate:"required"`         // 订单ID
	MerchantOrderId string       `json:"merchantOrderId" validate:"required"` // 商户订单ID
	RefundOrderId   int64        `json:"refundOrderId" validate:"required"`   // 退款订单ID
	Status          RefundStatus `json:"status" validate:"oneof=1 2 3 4"`     // 订单状态
	FailReason      string       `json:"failReason,omitempty"`                // 失败原因
}

// DisputeStatus 表示 payin 领域内的争议状态。
type DisputeStatus uint8

const (
	DisputeStatusReceived DisputeStatus = 1 // 收到争议
	DisputeStatusWin      DisputeStatus = 2 // 争议赢了
	DisputeStatusLose     DisputeStatus = 3 // 争议输了
)

type NotifyDisputeOrderPaylod struct {
	OrderID         int64         `json:"orderId" validate:"required"`         // 订单ID
	MerchantOrderId string        `json:"merchantOrderId" validate:"required"` // 商户订单ID
	DisputeOrderId  int64         `json:"disputeOrderId" validate:"required"`  // 争议订单ID
	Amount          Money         `json:"amount" validate:"required"`          // 争议金额
	Status          DisputeStatus `json:"status" validate:"oneof=1 2 3"`       // 订单状态
}

type NotifyAgreementStatus uint8

const (
	NotifyAgreementStatusApproved NotifyAgreementStatus = iota + 1
	NotifyAgreementStatusCancelled
)

type NotifyAgreementPaylod struct {
	OrderID         int64                 `json:"orderId" validate:"required"`         // 订单ID
	MerchantOrderId string                `json:"merchantOrderId" validate:"required"` // 商户订单ID
	AgreementID     int64                 `json:"agreementId" validate:"required"`     // 协议ID
	Email           string                `json:"email" validate:"required"`           // 用户邮箱
	UserID          string                `json:"userId" validate:"required"`          // 用户ID
	Status          NotifyAgreementStatus `json:"status" validate:"oneof=1 2"`         // 协议状态
}
