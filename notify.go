package payment

import (
	"crypto/ed25519"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"
)

type NotifyResp struct {
	MerchantID     string                    `json:"merchant_id"`               // 商户ID
	Tp             MerchantNotifyTp          `json:"tp"`                        // 通知类型
	PayinOrder     *NotifyOrderPayload       `json:"payin_order,omitempty"`     // 支付订单通知
	RefundOrder    *NotifyRefundOrderPayload `json:"refund_order,omitempty"`    // 退款订单通知
	DisputeOrder   *NotifyDisputePayload     `json:"dispute_order,omitempty"`   // 争议订单通知
	PayoutOrder    *NotifyOrderPayload       `json:"payout_order,omitempty"`    // 提现订单通知
	AgreementOrder *NotifyAgreementPayload   `json:"agreement_order,omitempty"` // 协议订单通知
	CardBind       *NotifyCardBindPayload    `json:"card_bind,omitempty"`       // 信用卡绑卡通知
	FraudOrder     *NotifyFraudPayload       `json:"fraud_order,omitempty"`     // 欺诈通知
}

type NotifyIdentity struct {
	MerchantID string           `json:"merchant_id"`
	OrderID    string           `json:"order_id"`
	Tp         MerchantNotifyTp `json:"tp"`
}

type NotifyReq struct {
	MerchantID   string
	OrderID      string
	Tp           MerchantNotifyTp
	Timestamp    string
	Nonce        string
	Signature    string
	SignatureAlg string
	PublicKey    ed25519.PublicKey
	Body         []byte
}

// Verify checks CallbackV2 (platform Ed25519). Hard cut: shared HMAC is not accepted.
func (n *NotifyReq) Verify() error {
	if n == nil {
		return errors.New("payment: notify request is required")
	}
	if len(n.PublicKey) != ed25519.PublicKeySize {
		return errors.New("payment: callback public key is required")
	}
	if strings.TrimSpace(n.Timestamp) == "" || strings.TrimSpace(n.Nonce) == "" || strings.TrimSpace(n.Signature) == "" {
		return errors.New("payment: notify signature headers are required")
	}
	alg := strings.TrimSpace(n.SignatureAlg)
	if alg == "" {
		return errors.New("payment: X-Signature-Alg is required")
	}
	if !strings.EqualFold(alg, SignatureAlgEd25519) {
		return fmt.Errorf("payment: unsupported signature alg %q", alg)
	}
	timestamp, err := strconv.ParseInt(strings.TrimSpace(n.Timestamp), 10, 64)
	if err != nil {
		return errors.New("payment: invalid notify timestamp")
	}
	now := time.Now()
	notifyAt := time.Unix(timestamp, 0)
	if notifyAt.After(now.Add(time.Duration(MaxFutureSkew) * time.Second)) {
		return errors.New("payment: notify timestamp in the future")
	}
	if now.Sub(notifyAt) > time.Duration(TimestampWindow)*time.Second {
		return errors.New("payment: notify timestamp expired")
	}
	if err := VerifyCallbackV2(n.PublicKey, strings.TrimSpace(n.Timestamp), strings.TrimSpace(n.Nonce), int(n.Tp), n.OrderID, n.Body, n.Signature); err != nil {
		return err
	}
	return nil
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
	case MerchantNotifyTpPayInCardBind:
		out.Tp = MerchantNotifyTpPayInCardBind
		var payload NotifyCardBindPayload
		if err := json.Unmarshal(n.Body, &payload); err != nil {
			return nil, err
		}
		out.CardBind = &payload
	case MerchantNotifyTpPayInFraud:
		out.Tp = MerchantNotifyTpPayInFraud
		var payload NotifyFraudPayload
		if err := json.Unmarshal(n.Body, &payload); err != nil {
			return nil, err
		}
		out.FraudOrder = &payload
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

func (n *Notify) ToNotifyReq(publicKey ed25519.PublicKey) (*NotifyReq, error) {
	identity, err := n.Identity()
	if err != nil {
		return nil, err
	}
	return &NotifyReq{
		MerchantID:   identity.MerchantID,
		OrderID:      identity.OrderID,
		Tp:           identity.Tp,
		Timestamp:    n.Header.Get(headerTimestamp),
		Nonce:        n.Header.Get(headerNonce),
		Signature:    n.Header.Get(headerSignature),
		SignatureAlg: n.Header.Get(headerSignatureAlg),
		PublicKey:    publicKey,
		Body:         n.Body,
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
	case MerchantNotifyTpPayInCardBind:
		if resp.CardBind != nil {
			payloadOrderID = resp.CardBind.OrderID
		}
	case MerchantNotifyTpPayInFraud:
		if resp.FraudOrder != nil {
			payloadOrderID = resp.FraudOrder.OrderID
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
	RefundStatusPending RefundStatus = 1 // 待处理
	RefundStatusSuccess RefundStatus = 2 // 成功
	RefundStatusFailed  RefundStatus = 3 // 失败（含拒绝）
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
	DisputeStatusCanceled DisputeStatus = 4 // 争议取消
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
	OrderID         string                `json:"orderId,omitempty"`         // 订单ID
	MerchantOrderId string                `json:"merchantOrderId,omitempty"` // 商户订单ID
	AgreementID     string                `json:"agreementId"`               // 协议ID
	Email           string                `json:"email"`                     // 用户邮箱
	UserID          string                `json:"userId"`                    // 用户ID
	Status          NotifyAgreementStatus `json:"status"`                    // 协议状态
}

type NotifyCardBindPayload struct {
	OrderID         string `json:"orderId"`
	MerchantOrderId string `json:"merchantOrderId"`
	UserID          string `json:"userId"`
	SourceID        string `json:"sourceId"`
	Last4           string `json:"last4,omitempty"`
	Bin             string `json:"bin,omitempty"`
	Scheme          string `json:"scheme,omitempty"`
	CardName        string `json:"cardName,omitempty"`
	ExpiryMonth     string `json:"expiryMonth,omitempty"`
	ExpiryYear      string `json:"expiryYear,omitempty"`
	Country         string `json:"country,omitempty"`
	Fingerprint     string `json:"fingerprint,omitempty"`
}

type NotifyFraudPayload struct {
	OrderID         string `json:"orderId"`
	MerchantOrderId string `json:"merchantOrderId"`
	UserID          string `json:"userId,omitempty"`
	Reason          string `json:"reason,omitempty"`
	Type            string `json:"type,omitempty"`
	Eci             string `json:"eci,omitempty"`
}
