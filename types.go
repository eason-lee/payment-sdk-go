package payment

import "errors"

const (
	headerMerchantID = "X-Merchant-Id"
	headerTimestamp  = "X-Timestamp"
	headerNonce      = "X-Nonce"
	headerSignature  = "X-Signature"
	headerNotifyTp   = "X-Notify-Tp"
	headerOrderID    = "X-Order-Id"
)

// PayMethod 支付方法
type PayMethod string

const (
	PayMethodPayPal PayMethod = "PayPal"
)

// PayMode 表示同一支付方式下的具体使用形态。
type PayMode string

const (
	PayModeCreditToken     PayMode = "CreditToken"     // 原生信用卡
	PayModeCreditSourceID  PayMode = "CreditSourceId"  // sourceId 信用卡绑卡支付
	PayModePayPal          PayMode = "PayPal"          // 直接拉起的
	PayModePayPalAgreement PayMode = "PayPalAgreement" // 授权绑定唯一ID的
	PayModeApplePayNative  PayMode = "ApplePayNative"  // 原生 applePay
	PayModeApplePayWeb     PayMode = "ApplePayWeb"     // 外跳 applePay
	PayModeGooglePayWeb    PayMode = "GooglePayWeb"    // 外跳 googlePay
	PayModeCashAppWeb      PayMode = "CashAppWeb"      // 外跳 cashApp
	PayModeSkrillWeb       PayMode = "SkrillWeb"       // 外跳 skrill
)

func (p PayMode) Valid() error {
	switch p {
	case PayModeCreditToken, PayModeCreditSourceID, PayModePayPal, PayModePayPalAgreement, PayModeApplePayNative, PayModeApplePayWeb, PayModeGooglePayWeb, PayModeCashAppWeb, PayModeSkrillWeb:
		return nil
	default:
		return errors.New("payment: invalid pay mode")
	}
}

// CurrencyTp 货币类型
type CurrencyTp string

const (
	CurrencyTpUSD CurrencyTp = "USD" // 美元
	CurrencyTpEUR CurrencyTp = "EUR" // 欧元
	CurrencyTpGBP CurrencyTp = "GBP" // 英镑
)

const (
	productionBaseURL = "http://192.168.1.171"   // TODO 生产环境地址待确认
	sandboxBaseURL    = "http://192.168.160.191" // TODO 沙箱环境地址待确认
	testBaseURL       = "http://127.0.0.1:8083"
)

type EnvType string

const (
	EnvProduction EnvType = "production"
	EnvSandbox    EnvType = "sandbox"
	EnvTest       EnvType = "test"
)

// Production returns production merchant gateway environment.
func (p EnvType) GetBaseURL() string {
	switch p {
	case EnvProduction:
		return productionBaseURL
	case EnvSandbox:
		return sandboxBaseURL
	case EnvTest:
		return testBaseURL
	default:
		return ""
	}
}

func Production() EnvType {
	return EnvProduction
}

func Sandbox() EnvType {
	return EnvSandbox
}

// MerchantNotifyTp 表示商户通知回调类型。
type MerchantNotifyTp uint8

const (
	// MerchantNotifyTpPayIn 表示代收订单通知。
	MerchantNotifyTpPayIn MerchantNotifyTp = 1
	// MerchantNotifyTpPayInRefund 表示代收退款通知。
	MerchantNotifyTpPayInRefund MerchantNotifyTp = 2
	// MerchantNotifyTpPayInDispute 表示代收订单争议通知。
	MerchantNotifyTpPayInDispute MerchantNotifyTp = 3
	// MerchantNotifyTpPayOut 表示代付订单通知。
	MerchantNotifyTpPayOut MerchantNotifyTp = 4
	// MerchantNotifyTpPayInAgreement 表示代收 PayPal 协议签约通知。
	MerchantNotifyTpPayInAgreement MerchantNotifyTp = 5
)

// PayinOrderStatus 是商户侧可见的代收订单状态。
type PayinOrderStatus int

const (
	// PayinOrderStatusUnspecified 表示未知代收订单状态，正常业务不应依赖该状态。
	PayinOrderStatusUnspecified PayinOrderStatus = iota
	// PayinOrderStatusProcessing 表示代收订单处理中，请等待通知或继续查询。
	PayinOrderStatusProcessing
	// PayinOrderStatusSuccess 表示代收订单支付成功。
	PayinOrderStatusSuccess
	// PayinOrderStatusFailed 表示代收订单支付失败。
	PayinOrderStatusFailed
	// PayinOrderStatusRefunding 表示代收订单退款处理中。
	PayinOrderStatusRefunding
	// PayinOrderStatusRefunded 表示代收订单已退款。
	PayinOrderStatusRefunded
	// PayinOrderStatusRefundFailed 表示代收订单退款失败或被拒绝。
	PayinOrderStatusRefundFailed
	// PayinOrderStatusCanceled 表示代收订单已取消。
	PayinOrderStatusCanceled
)

// PayoutOrderStatus 是商户侧可见的代付订单状态。
type PayoutOrderStatus int

const (
	// PayoutOrderStatusUnspecified 表示未知代付订单状态，正常业务不应依赖该状态。
	PayoutOrderStatusUnspecified PayoutOrderStatus = iota
	// PayoutOrderStatusProcessing 表示代付订单处理中，请等待通知或继续查询。
	PayoutOrderStatusProcessing
	// PayoutOrderStatusSuccess 表示代付订单代付成功。
	PayoutOrderStatusSuccess
	// PayoutOrderStatusFailed 表示代付订单代付失败。
	PayoutOrderStatusFailed
)

// Money 金额，单位为分
type Money int64

func NewMoney(f int64) Money {
	return Money(f)
}
