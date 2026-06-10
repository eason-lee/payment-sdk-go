package payment

import "errors"

const (
	headerMerchantID = "X-Merchant-Id"
	headerTimestamp  = "X-Timestamp"
	headerNonce      = "X-Nonce"
	headerSignature  = "X-Signature"
	headerNotifyTp   = "X-Notify-Tp"
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
)

const (
	productionBaseURL = "http://192.168.1.171"
	sandboxBaseURL    = "http://127.0.0.1:8080"
)

type EnvType string

const (
	EnvProduction EnvType = "production"
	EnvSandbox    EnvType = "sandbox"
)

// Production returns production merchant gateway environment.
func (p EnvType) GetBaseURL() string {
	switch p {
	case EnvProduction:
		return productionBaseURL
	case EnvSandbox:
		return sandboxBaseURL
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

// Money 金额，单位为分
type Money int64

func NewMoney(f int64) Money {
	return Money(f)
}
