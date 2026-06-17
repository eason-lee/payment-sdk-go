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
	"time"
)

const notifyMaxClockSkew = 5 * time.Minute

type NotifyResp struct {
	MerchantID     string                    `json:"merchant_id"`               // 商户ID
	Tp             MerchantNotifyTp          `json:"tp"`                        // 通知类型
	PayinOrder     *NotifyOrderPayload       `json:"payin_order,omitempty"`     // 支付订单通知
	RefundOrder    *NotifyRefundOrderPayload `json:"refund_order,omitempty"`    // 退款订单通知
	DisputeOrder   *NotifyDisputePayload     `json:"dispute_order,omitempty"`   // 争议订单通知
	PayoutOrder    *NotifyOrderPayload       `json:"payout_order,omitempty"`    // 提现订单通知
	AgreementOrder *NotifyAgreementPayload   `json:"agreement_order,omitempty"` // 协议订单通知
}

type NotifyIdentity struct {
	MerchantID string           `json:"merchant_id"`
	OrderID    string           `json:"order_id"`
	Tp         MerchantNotifyTp `json:"tp"`
}

type NotifyReq struct {
	MerchantID string
	OrderID    string
	Tp         MerchantNotifyTp
	Timestamp  string
	Nonce      string
	Signature  string
	Secret     string
	Body       []byte
}

func (n *NotifyReq) Verify() bool {
	if strings.TrimSpace(n.Secret) == "" || strings.TrimSpace(n.Timestamp) == "" || strings.TrimSpace(n.Nonce) == "" || strings.TrimSpace(n.Signature) == "" {
		return false
	}
	timestamp, err := strconv.ParseInt(strings.TrimSpace(n.Timestamp), 10, 64)
	if err != nil {
		return false
	}
	now := time.Now()
	notifyAt := time.Unix(timestamp, 0)
	if notifyAt.Before(now.Add(-notifyMaxClockSkew)) || notifyAt.After(now.Add(notifyMaxClockSkew)) {
		return false
	}

	mac := hmac.New(sha256.New, []byte(n.Secret))
	mac.Write([]byte(n.Timestamp))
	mac.Write([]byte("\n"))
	mac.Write([]byte(n.Nonce))
	mac.Write([]byte("\n"))
	mac.Write([]byte(strconv.Itoa(int(n.Tp))))
	mac.Write([]byte("\n"))
	mac.Write([]byte(n.OrderID))
	mac.Write([]byte("\n"))
	mac.Write(n.Body)
	got, err := hex.DecodeString(strings.TrimSpace(n.Signature))
	if err != nil {
		return false
	}
	return hmac.Equal(got, mac.Sum(nil))
}

func (n *NotifyReq) ToNotifyResp() (*NotifyResp, error) {

	out := &NotifyResp{
		MerchantID: n.MerchantID,
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
	Header http.Header
	Body   []byte
}

func (n *Notify) Identity() (*NotifyIdentity, error) {
	if n == nil {
		return nil, errors.New("payment: notify is required")
	}
	mid := strings.TrimSpace(n.Header.Get(headerMerchantID))
	if mid == "" {
		return nil, errors.New("payment: invalid notify merchant id")
	}
	orderID := strings.TrimSpace(n.Header.Get(headerOrderID))
	if orderID == "" {
		return nil, errors.New("payment: invalid notify order id")
	}
	notifyTp, err := strconv.Atoi(strings.TrimSpace(n.Header.Get(headerNotifyTp)))
	if err != nil {
		return nil, err
	}
	return &NotifyIdentity{
		MerchantID: mid,
		OrderID:    orderID,
		Tp:         MerchantNotifyTp(notifyTp),
	}, nil
}

func (n *Notify) ToNotifyReq(secret string) (*NotifyReq, error) {
	identity, err := n.Identity()
	if err != nil {
		return nil, err
	}
	return &NotifyReq{
		MerchantID: identity.MerchantID,
		OrderID:    identity.OrderID,
		Tp:         identity.Tp,
		Timestamp:  n.Header.Get(headerTimestamp),
		Nonce:      n.Header.Get(headerNonce),
		Signature:  n.Header.Get(headerSignature),
		Secret:     secret,
		Body:       n.Body,
	}, nil
}

func (n *NotifyReq) ValidatePayloadOrderID(resp *NotifyResp) error {
	if resp == nil {
		return errors.New("payment: notify response is required")
	}
	var payloadOrderID string
	switch n.Tp {
	case MerchantNotifyTpPayIn:
		if resp.PayinOrder != nil {
			payloadOrderID = resp.PayinOrder.OrderID
		}
	case MerchantNotifyTpPayOut:
		if resp.PayoutOrder != nil {
			payloadOrderID = resp.PayoutOrder.OrderID
		}
	case MerchantNotifyTpPayInRefund:
		if resp.RefundOrder != nil {
			payloadOrderID = resp.RefundOrder.OrderID
		}
	case MerchantNotifyTpPayInDispute:
		if resp.DisputeOrder != nil {
			payloadOrderID = resp.DisputeOrder.OrderID
		}
	case MerchantNotifyTpPayInAgreement:
		if resp.AgreementOrder != nil {
			payloadOrderID = resp.AgreementOrder.OrderID
		}
	}
	if payloadOrderID != "" && payloadOrderID != n.OrderID {
		return errors.New("payment: order id in header not equal to order id in body")
	}
	return nil
}

type NotifyOrderStatus uint8

const (
	NotifyOrderStatusSuccess NotifyOrderStatus = iota + 1
	NotifyOrderStatusFailed
)

// NotifyOrderPayload 支付订单通知负载
type NotifyOrderPayload struct {
	OrderID         string            `json:"orderId"`              // 订单ID
	MerchantOrderId string            `json:"merchantOrderId"`      // 商户订单ID
	Status          NotifyOrderStatus `json:"status"`               // 订单状态
	Fee             Money             `json:"fee"`                  // 手续费
	FailReason      string            `json:"failReason,omitempty"` // 失败原因
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
	OrderID         string       `json:"orderId"`
	MerchantOrderId string       `json:"merchantOrderId"`
	RefundOrderId   string       `json:"refundOrderId"`
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
	OrderID         string        `json:"orderId"`
	MerchantOrderId string        `json:"merchantOrderId"`
	DisputeOrderId  string        `json:"disputeOrderId"`
	Amount          Money         `json:"amount"`
	Status          DisputeStatus `json:"status"`
}

type NotifyAgreementStatus uint8

const (
	NotifyAgreementStatusApproved NotifyAgreementStatus = iota + 1
	NotifyAgreementStatusCancelled
)

type NotifyAgreementPayload struct {
	OrderID         string                `json:"orderId"`         // 订单ID
	MerchantOrderId string                `json:"merchantOrderId"` // 商户订单ID
	AgreementID     string                `json:"agreementId"`     // 同意ID
	Email           string                `json:"email"`           // 用户邮箱
	UserID          string                `json:"userId"`          // 用户ID
	Status          NotifyAgreementStatus `json:"status"`          // 同意状态
}
