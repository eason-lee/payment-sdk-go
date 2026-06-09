package payment

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

const (
	headerMerchantID = "X-Merchant-Id"
	headerTimestamp  = "X-Timestamp"
	headerNonce      = "X-Nonce"
	headerSignature  = "X-Signature"
)

type ClientImpl struct {
	baseURL string
}

func (c *ClientImpl) CreatePayin(ctx context.Context, req CreatePayinReq) (*CreatePayinResp, error) {
	if err := req.Valid(); err != nil {
		return nil, err
	}
	
	var out CreatePayinResp
	if err := c.doJSON(ctx, req.Merchant, http.MethodPost, "/api/payin/order", req, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

func (c *ClientImpl) GetPayin(ctx context.Context, merchant Merchant, orderID int64) (*PayinOrderResp, error) {
	var out PayinOrderResp
	if err := c.doJSON(ctx, merchant, http.MethodGet, fmt.Sprintf("/api/payin/order/%d", orderID), nil, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

func (c *ClientImpl) RefundPayin(ctx context.Context, merchant Merchant, orderID int64, req RefundPayinReq) error {
	return c.doJSON(ctx, merchant, http.MethodPost, fmt.Sprintf("/api/payin/order/%d/refund", orderID), req, nil)
}

func (c *ClientImpl) CreatePayout(ctx context.Context, req CreatePayoutReq) (*CreatePayoutResp, error) {
	var out CreatePayoutResp
	if err := c.doJSON(ctx, req.Merchant, http.MethodPost, "/api/payout/order", req, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

func (c *ClientImpl) GetPayout(ctx context.Context, merchant Merchant, orderID int64) (*PayoutOrderResp, error) {
	var out PayoutOrderResp
	if err := c.doJSON(ctx, merchant, http.MethodGet, fmt.Sprintf("/api/payout/order/%d", orderID), nil, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

func (c *ClientImpl) doJSON(ctx context.Context, merchant Merchant, method string, path string, body any, out any) error {
	merchantID := strings.TrimSpace(merchant.ID)
	if merchantID == "" {
		return errors.New("payment: merchant id is required")
	}
	if strings.TrimSpace(merchant.Secret) == "" {
		return errors.New("payment: merchant secret is required")
	}

	var rawBody []byte
	var err error
	if body != nil {
		rawBody, err = json.Marshal(body)
		if err != nil {
			return fmt.Errorf("payment: marshal request body: %w", err)
		}
	}

	req, err := http.NewRequestWithContext(ctx, method, c.baseURL+path, bytes.NewReader(rawBody))
	if err != nil {
		return err
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	req.Header.Set("Accept", "application/json")

	timestamp := time.Now().Format(time.RFC3339)
	nonce, err := c.generateNonce()
	if err != nil {
		return fmt.Errorf("payment: generate nonce: %w", err)
	}
	signature := c.signRequest(merchant.Secret, signRequestInput{
		Method:    method,
		Path:      path,
		Timestamp: timestamp,
		Nonce:     nonce,
		Body:      rawBody,
	})

	req.Header.Set(headerMerchantID, merchantID)
	req.Header.Set(headerTimestamp, timestamp)
	req.Header.Set(headerNonce, nonce)
	req.Header.Set(headerSignature, signature)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("payment: read response body: %w", err)
	}
	return c.decodeResponse(resp.StatusCode, respBody, out)
}

type responseEnvelope struct {
	Message string          `json:"message"`
	Data    json.RawMessage `json:"data"`
}

func (c *ClientImpl) decodeResponse(statusCode int, body []byte, out any) error {
	var envelope responseEnvelope
	if len(body) > 0 {
		if err := json.Unmarshal(body, &envelope); err != nil {
			if statusCode < 200 || statusCode >= 300 {
				return &APIError{StatusCode: statusCode, Body: body}
			}
			return fmt.Errorf("payment: decode response envelope: %w", err)
		}
	}

	if statusCode < 200 || statusCode >= 300 {
		return &APIError{
			StatusCode: statusCode,
			Message:    strings.TrimSpace(envelope.Message),
			Body:       body,
		}
	}
	if out == nil || len(envelope.Data) == 0 || string(envelope.Data) == "null" {
		return nil
	}
	if err := json.Unmarshal(envelope.Data, out); err != nil {
		return fmt.Errorf("payment: decode response data: %w", err)
	}
	return nil
}

func (c *ClientImpl) generateNonce() (string, error) {
	const letters = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	const length = 32

	raw := make([]byte, length)
	if _, err := rand.Read(raw); err != nil {
		return "", err
	}
	for i := range raw {
		raw[i] = letters[int(raw[i])%len(letters)]
	}
	return string(raw), nil
}

type signRequestInput struct {
	Method    string
	Path      string
	Timestamp string
	Nonce     string
	Body      []byte
}

func (c *ClientImpl) signRequest(secret string, in signRequestInput) string {
	var payload strings.Builder
	payload.WriteString(in.Method)
	payload.WriteString("\n")
	payload.WriteString(in.Path)
	payload.WriteString("\n")
	payload.WriteString(in.Timestamp)
	payload.WriteString("\n")
	payload.WriteString(in.Nonce)
	payload.WriteString("\n")
	payload.Write(in.Body)

	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte(payload.String()))
	return hex.EncodeToString(mac.Sum(nil))
}
