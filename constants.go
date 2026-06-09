package payment

const (
	headerMerchantID = "X-Merchant-Id"
	headerTimestamp  = "X-Timestamp"
	headerNonce      = "X-Nonce"
	headerSignature  = "X-Signature"

	defaultUserAgent = "payment-sdk-go"
)

const (
	PayMethodGCash  = "GCASH"
	PayMethodPayPal = "PAYPAL"

	PayModePayPal          = "PayPal"
	PayModePayPalAgreement = "PayPalAgreement"
)
