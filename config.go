package payment

import (
	"net/http"
	"time"
)

// Config configures a merchant gateway client.
type Config struct {
	BaseURL    string
	MerchantID string
	Secret     string
	HTTPClient *http.Client
	UserAgent  string
	Now        func() time.Time
	Nonce      func() (string, error)
}

func (c Config) httpClient() *http.Client {
	if c.HTTPClient != nil {
		return c.HTTPClient
	}
	return http.DefaultClient
}

func (c Config) now() time.Time {
	if c.Now != nil {
		return c.Now()
	}
	return time.Now()
}

func (c Config) nonce() (string, error) {
	if c.Nonce != nil {
		return c.Nonce()
	}
	return GenerateNonce()
}
