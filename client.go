package payment

import (
	"context"
	"net/http"
	"time"
)

// Client calls payment merchant gateway APIs.
type Client interface {
	CreatePayin(ctx context.Context, req *CreatePayinReq) (*CreatePayinResp, error)
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
	MerchantOrderID string      `json:"merchant_order_id" validate:"required"` // 商户订单ID
	Amount          int64       `json:"amount" validate:"gt=0"`                // 金额, 单位分
	Currency        CurrencyTp  `json:"currency" validate:"oneof=USD"`         // 货币类型
	PayMethod       PayMethod   `json:"pay_method" validate:"oneof=PayPal"`    // 支付方式
	PayMode         PayMode     `json:"pay_mode" validate:"required"`          // 支付模式
	User            *User       `json:"user,omitempty"`                        // 用户信息
	PayPal          *PayPal     `json:"paypal,omitempty"`                      // PayPal支付信息
	ApplePay        *ApplePay   `json:"apple_pay,omitempty"`                   // ApplePay支付信息
	CreditCard      *CreditCard `json:"credit_card,omitempty"`                 // 信用卡支付信息
	GooglePay       *GooglePay  `json:"google_pay,omitempty"`                  // GooglePay支付信息
}

func (r CreatePayinReq) Valid() error {
	if err := r.PayMode.Valid(); err != nil {
		return err
	}
	return ValidStruct(r)
}

type User struct {
	ID      string `json:"id" validate:"required"`       // 用户ID
	AppName string `json:"app_name" validate:"required"` // 应用名称
	Name    string `json:"name,omitempty"`               // 用户姓名
	Email   string `json:"email,omitempty"`              // 用户邮箱
	Phone   string `json:"phone,omitempty"`              // 用户手机号
}

type PayPal struct {
	Email string `json:"email"` // PayPal邮箱
}

type ApplePay struct {
	AppleToken string `json:"apple_token,omitempty"` // ApplePay支付令牌
	Country    string `json:"country,omitempty"`
	City       string `json:"city,omitempty"`
	State      string `json:"state,omitempty"`
}

type CreditCard struct {
	SourceID    string `json:"source_id,omitempty"`
	CreditToken string `json:"credit_token,omitempty"`
	Address     string `json:"address,omitempty"`
	ZipCode     string `json:"zip_code,omitempty"`
	Country     string `json:"country,omitempty"`
	City        string `json:"city,omitempty"`
	State       string `json:"state,omitempty"`
}

type GooglePay struct {
	Signature       string `json:"signature,omitempty"`
	SignedMessage   string `json:"signed_message,omitempty"`
	ProtocolVersion string `json:"protocol_version,omitempty"`
}

type CreatePayinResp struct {
	OrderID string `json:"order_id"`
	Link    string `json:"link"`
}

type PayinOrderResp struct {
	OrderID         string           `json:"order_id"`
	MerchantOrderID string           `json:"merchant_order_id"`
	Status          PayinOrderStatus `json:"status"`
	FailReason      string           `json:"fail_reason,omitempty"`
	Amount          int64            `json:"amount"`
	RefundedAmount  int64            `json:"refunded_amount,omitempty"`
	Currency        CurrencyTp       `json:"currency"`
	PayMethod       PayMethod        `json:"pay_method"`
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
	Currency        CurrencyTp `json:"currency" validate:"oneof=USD"`         // 货币类型
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
