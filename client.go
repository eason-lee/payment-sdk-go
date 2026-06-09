package payment

import (
	"context"
	"errors"
	"net/http"
	"net/url"
	"strings"
)

// Client calls payment merchant gateway APIs.
type Client struct {
	baseURL    *url.URL
	merchantID string
	secret     string
	httpClient *http.Client
	userAgent  string
	now        func() string
	nonce      func() (string, error)
}

// NewClient creates a merchant gateway client.
func NewClient(config Config) *Client {
	client, err := NewClientWithError(config)
	if err != nil {
		panic(err)
	}
	return client
}

// NewClientWithError creates a merchant gateway client and reports config errors.
func NewClientWithError(config Config) (*Client, error) {
	if strings.TrimSpace(config.BaseURL) == "" {
		return nil, errors.New("payment: base url is required")
	}
	if strings.TrimSpace(config.MerchantID) == "" {
		return nil, errors.New("payment: merchant id is required")
	}
	if strings.TrimSpace(config.Secret) == "" {
		return nil, errors.New("payment: secret is required")
	}
	baseURL, err := url.Parse(strings.TrimRight(config.BaseURL, "/"))
	if err != nil {
		return nil, err
	}
	if baseURL.Scheme == "" || baseURL.Host == "" {
		return nil, errors.New("payment: base url must include scheme and host")
	}

	userAgent := strings.TrimSpace(config.UserAgent)
	if userAgent == "" {
		userAgent = defaultUserAgent
	}

	return &Client{
		baseURL:    baseURL,
		merchantID: strings.TrimSpace(config.MerchantID),
		secret:     config.Secret,
		httpClient: config.httpClient(),
		userAgent:  userAgent,
		now: func() string {
			return formatUnix(config.now())
		},
		nonce: config.nonce,
	}, nil
}

func (c *Client) do(ctx context.Context, method string, path string, body any, out any) error {
	return c.doJSON(ctx, method, path, body, out)
}
