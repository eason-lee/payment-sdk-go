package payment


// PayMethod 支付方法
type PayMethod string

const (
	PayMethodPayPal     PayMethod = "PAYPAL"
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

// CurrencyTp 货币类型
type CurrencyTp string

const (
	CurrencyTpUSD CurrencyTp = "USD" // 美元
)


const (
	productionBaseURL = "http://192.168.1.171"
	sandboxBaseURL    = "http://192.168.1.171"
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
