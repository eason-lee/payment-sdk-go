package payment

import (
	"context"
	"errors"
	"net/http"
	"time"
)

// Client calls payment merchant gateway APIs.
type Client interface {
	CreatePayin(ctx context.Context, req *CreatePayinReq) (*CreatePayinResp, error)
	// SubmitCreditFlow 提交 CreditFlow payment session：Forter 前置风控后由 payment 调用 Checkout submit。
	// 必须由 Merchant 服务端调用；前端只通过 Checkout Flow handleSubmit 拿到 session_data 与非敏感卡片快照。
	SubmitCreditFlow(ctx context.Context, req *SubmitCreditFlowReq) (*SubmitCreditFlowResp, error)
	GetPayin(ctx context.Context, req *GetPayinReq) (*PayinOrderResp, error)
	RefundPayin(ctx context.Context, req RefundPayinReq) error
	CreatePayout(ctx context.Context, req *CreatePayoutReq) (*CreatePayoutResp, error)
	GetPayout(ctx context.Context, req *GetPayoutReq) (*PayoutOrderResp, error)
	NotifyClient
}

type NotifyClient interface {
	GetNotifyIdentity(notify *Notify) (*NotifyIdentity, error)
	ParseNotify(ctx context.Context, secret string, notify *Notify) (*NotifyResp, error)
	NotifySuccess(w http.ResponseWriter)
	NotifyFailed(w http.ResponseWriter)
}

type GetPayinReq struct {
	Merchant
	OrderID string `json:"order_id" validate:"required"`
}

func (r GetPayinReq) Valid() error {
	return ValidStruct(r)
}

// GetPayoutReq 获取ayout订单详情请求参数
type GetPayoutReq struct {
	Merchant
	OrderID string `json:"order_id" validate:"required"` // 订单ID
}

func (r GetPayoutReq) Valid() error {
	return ValidStruct(r)
}

func NewClient(env EnvType) *ClientImpl {
	transport := http.DefaultTransport.(*http.Transport).Clone()
	transport.Proxy = nil

	return &ClientImpl{
		baseURL: env.GetBaseURL(),
		httpClient: &http.Client{
			Transport: transport,
			Timeout:   15 * time.Second,
		},
	}
}

type Merchant struct {
	ID     string `json:"id" validate:"gt=0"`
	Secret string `json:"-" validate:"required"`
}

type CreatePayinReq struct {
	Merchant
	MerchantOrderID string      `json:"merchant_order_id" validate:"required,max=64"`                                    // 商户订单ID
	Amount          int64       `json:"amount" validate:"gt=0"`                                                          // 金额, 单位分
	Currency        CurrencyTp  `json:"currency" validate:"oneof=USD EUR GBP"`                                           // 货币类型
	PayMethod       PayMethod   `json:"pay_method" validate:"oneof=PayPal ApplePay GooglePay CreditCard CashApp Skrill"` // 支付方式
	PayMode         PayMode     `json:"pay_mode" validate:"required"`                                                    // 支付模式
	User            *User       `json:"user,omitempty"`                                                                  // 用户信息
	Address         *Address    `json:"address" validate:"required"`                                                     // 账单/支付地址；支付国放 Country
	PayPal          *PayPal     `json:"paypal,omitempty"`                                                                // PayPal 支付信息
	CreditCard      *CreditCard `json:"credit_card,omitempty"`                                                           // 信用卡凭证；CreditToken/CreditSourceId 必填
	ApplePay        *ApplePay   `json:"apple_pay,omitempty"`                                                             // ApplePay 支付信息
	GooglePay       *GooglePay  `json:"google_pay,omitempty"`                                                            // GooglePay 支付信息
}

// Address 账单/支付地址。支付国使用 Country。
type Address struct {
	Country string `json:"country,omitempty"`
	Address string `json:"address,omitempty" validate:"omitempty,max=255"`
	State   string `json:"state,omitempty" validate:"omitempty,max=64"`
	City    string `json:"city,omitempty" validate:"omitempty,max=64"`
	Zip     string `json:"zip,omitempty" validate:"omitempty,max=32"`
}

func (r CreatePayinReq) Valid() error {
	if err := ValidStruct(r); err != nil {
		return err
	}
	if err := r.PayMode.Valid(r.PayMethod); err != nil {
		return err
	}
	if r.PayMode == PayModeCreditFlow {
		if r.Address.State == "" || r.Address.City == "" {
			return errors.New("address.state and address.city are required")
		}
	}
	switch r.PayMode {
	case PayModeCreditToken:
		if r.CreditCard == nil || r.CreditCard.Token == "" {
			return errors.New("credit_card.token is required")
		}
	case PayModeCreditSourceID:
		if r.CreditCard == nil || r.CreditCard.SourceID == "" {
			return errors.New("credit_card.source_id is required")
		}
	}
	return nil
}

type User struct {
	ID      string `json:"id" validate:"required"`              // 用户ID
	AppName string `json:"app_name" validate:"required,max=50"` // 应用名称
	Name    string `json:"name,omitempty"`                      // 用户姓名
	Email   string `json:"email,omitempty"`                     // 用户邮箱
	Phone   string `json:"phone,omitempty"`                     // 用户手机号
}

type PayPal struct {
	Email string `json:"email"` // PayPal邮箱
}

// ApplePay Apple Pay 支付方式参数。地址走顶层 address。
type ApplePay struct {
	AppleToken string `json:"apple_token,omitempty"` // ApplePay支付令牌
}

// CreditCard 建单信用卡凭证。Token 路径可选传入 tokenize 回包中的卡面信息；地址走顶层 address。
type CreditCard struct {
	SourceID    string `json:"source_id,omitempty"`
	Token       string `json:"token,omitempty"`
	CardName    string `json:"card_name,omitempty" validate:"omitempty,max=128"`
	Last4       string `json:"last4,omitempty" validate:"omitempty,max=4"`
	ExpiryMonth string `json:"expiry_month,omitempty" validate:"omitempty,max=2"`
	ExpiryYear  string `json:"expiry_year,omitempty" validate:"omitempty,max=4"`
}

// CardRiskSnapshot 是非敏感卡片风险快照，用于 Forter 前置风控。
// 禁止包含完整卡号（PAN）或 CVV。字段名与 merchant CardRiskInfo 一致。
type CardRiskSnapshot struct {
	CardName    string `json:"card_name,omitempty"`
	Bin         string `json:"bin,omitempty"`
	Last4       string `json:"last4,omitempty"`
	ExpiryMonth string `json:"expiry_month,omitempty"`
	ExpiryYear  string `json:"expiry_year,omitempty"`
	Scheme      string `json:"scheme,omitempty"`
	Country     string `json:"country,omitempty"`
}

// SubmitCreditFlowReq 是 CreditFlow 第二阶段服务端提交请求。
type SubmitCreditFlowReq struct {
	Merchant
	// OrderID 是 CreatePayin 返回的 payment 订单 ID。
	OrderID string `json:"-" validate:"required"`
	// SessionID 必须与建单返回的 PaymentSession.ID 一致。
	SessionID string `json:"session_id" validate:"required"`
	// SessionData 来自 Checkout Flow handleSubmit，不透明，不要写入日志。
	SessionData string `json:"session_data" validate:"required"`
	// Card 非敏感卡片风险快照；禁止 PAN/CVV。
	Card *CardRiskSnapshot `json:"card" validate:"required"`
}

func (r SubmitCreditFlowReq) Valid() error {
	if r.Card == nil {
		return errors.New("card risk snapshot is required")
	}
	return ValidStruct(r)
}

// SubmitCreditFlowResp 是项目内归一化提交结果，不包含 Checkout 原始 DTO。
type SubmitCreditFlowResp struct {
	OrderID    string `json:"order_id"`
	Status     string `json:"status"`
	PaymentID  string `json:"payment_id,omitempty"`
	ActionType string `json:"action_type,omitempty"`
	ActionURL  string `json:"action_url,omitempty"`
	FailReason string `json:"fail_reason,omitempty"`
}

type GooglePay struct {
	Signature       string `json:"signature,omitempty"`
	SignedMessage   string `json:"signed_message,omitempty"`
	ProtocolVersion string `json:"protocol_version,omitempty"`
}

type CreatePayinResp struct {
	OrderID        string                  `json:"order_id"`
	Link           string                  `json:"link"`
	PaymentSession *CheckoutPaymentSession `json:"payment_session,omitempty"`
}

type CheckoutPaymentSession struct {
	ID     string `json:"id"`
	Token  string `json:"token"`
	Secret string `json:"secret"`
}

type PayinOrderResp struct {
	OrderID         string           `json:"order_id"`
	MerchantOrderID string           `json:"merchant_order_id"`
	Status          PayinOrderStatus `json:"status"`
	FailReason      string           `json:"fail_reason,omitempty"`
	Amount          int64            `json:"amount"`
	HasRefund       bool             `json:"has_refund,omitempty"`
	Currency        CurrencyTp       `json:"currency"`
	PayMethod       PayMethod        `json:"pay_method,omitempty"`
	Refund          *PayinRefund     `json:"refund,omitempty"`
	Dispute         *PayinDispute    `json:"dispute,omitempty"`
}

// PayinRefund 是查询代收订单时返回的关联退款。
type PayinRefund struct {
	RefundOrderID string `json:"refund_order_id"`
	Status        int32  `json:"status"`
	OutRefundID   string `json:"out_refund_id,omitempty"`
	FailReason    string `json:"fail_reason,omitempty"`
	Amount        int64  `json:"amount"`
}

// PayinDispute 是查询代收订单时返回的关联争议。
type PayinDispute struct {
	DisputeID    string     `json:"dispute_id"`
	OutDisputeID string     `json:"out_dispute_id,omitempty"`
	Status       int32      `json:"status"`
	Reason       string     `json:"reason,omitempty"`
	Amount       int64      `json:"amount"`
	Currency     CurrencyTp `json:"currency,omitempty"`
}

type RefundPayinReq struct {
	Merchant
	OrderID string `json:"order_id" validate:"required"` // 订单ID
	Amount  int64  `json:"amount" validate:"gt=0"`       // 金额, 单位分
}

func (r RefundPayinReq) Valid() error {
	return ValidStruct(r)
}

type CreatePayoutReq struct {
	Merchant
	MerchantOrderID string     `json:"merchant_order_id" validate:"required"` // 商户订单ID
	Amount          int64      `json:"amount" validate:"gt=0"`                // 金额, 单位分
	Currency        CurrencyTp `json:"currency" validate:"oneof=USD EUR GBP"` // 货币类型
	PayMethod       PayMethod  `json:"pay_method" validate:"oneof=PayPal"`    // 支付方式
	Account         string     `json:"account" validate:"required"`           // 支付账号
	User            *User      `json:"user,omitempty"`                        // 用户信息
}

func (r CreatePayoutReq) Valid() error {
	return ValidStruct(r)
}

// CreatePayoutResp 创建payout订单响应参数
type CreatePayoutResp struct {
	OrderID string `json:"order_id"` // 订单ID
}

type PayoutOrderResp struct {
	OrderID         string            `json:"order_id"`          // 订单ID
	MerchantOrderID string            `json:"merchant_order_id"` // 商户订单ID
	Status          PayoutOrderStatus `json:"status"`            // 代付订单状态
	PayMethod       PayMethod         `json:"pay_method,omitempty"`
	FailReason      string            `json:"fail_reason,omitempty"`
	Amount          int64             `json:"amount"`
	Currency        CurrencyTp        `json:"currency"`
}
