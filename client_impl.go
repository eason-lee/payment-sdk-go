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
	"net/url"
	"strings"
	"time"
)

type ClientImpl struct {
	baseURL    string
	httpClient *http.Client
}

func (c *ClientImpl) CreatePayin(ctx context.Context, req *CreatePayinReq) (*CreatePayinResp, error) {
	if err := req.Valid(); err != nil {
		return nil, err
	}

	var out CreatePayinResp
	if err := c.doJSON(ctx, req.Merchant, http.MethodPost, "/api/payin/order", req, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

func (c *ClientImpl) SubmitCreditFlow(ctx context.Context, req *SubmitCreditFlowReq) (*SubmitCreditFlowResp, error) {
	if err := req.Valid(); err != nil {
		return nil, err
	}

	path := fmt.Sprintf("/api/payin/order/s/%s/checkout/submit", url.PathEscape(req.OrderID))
	// body 不含 order_id / merchant 密钥；merchant 只用于签名头。
	body := struct {
		SessionID   string            `json:"session_id"`
		SessionData string            `json:"session_data"`
		Card        *CardRiskSnapshot `json:"card"`
		IP          string            `json:"ip,omitempty"`
		UserAgent   string            `json:"user_agent,omitempty"`
		RiskToken   string            `json:"risk_token,omitempty"`
	}{
		SessionID:   strings.TrimSpace(req.SessionID),
		SessionData: req.SessionData,
		Card:        req.Card,
		IP:          strings.TrimSpace(req.IP),
		UserAgent:   strings.TrimSpace(req.UserAgent),
		RiskToken:   strings.TrimSpace(req.RiskToken),
	}

	var out SubmitCreditFlowResp
	if err := c.doJSON(ctx, req.Merchant, http.MethodPost, path, body, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

func (c *ClientImpl) GetPayin(ctx context.Context, req *GetPayinReq) (*PayinOrderResp, error) {
	if err := req.Valid(); err != nil {
		return nil, err
	}

	var out PayinOrderResp
	if err := c.doJSON(ctx, req.Merchant, http.MethodGet, fmt.Sprintf("/api/payin/order/s/%s", url.PathEscape(req.OrderID)), nil, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

func (c *ClientImpl) RefundPayin(ctx context.Context, req RefundPayinReq) error {
	if err := req.Valid(); err != nil {
		return err
	}

	return c.doJSON(ctx, req.Merchant, http.MethodPost, fmt.Sprintf("/api/payin/order/s/%s/refund", url.PathEscape(req.OrderID)), req, nil)
}

func (c *ClientImpl) AppealDispute(ctx context.Context, req *AppealDisputeReq) error {
	if req == nil {
		return errors.New("payment: appeal request is required")
	}
	if err := req.Valid(); err != nil {
		return err
	}
	body := struct {
		FileURL string `json:"file_url"`
		Notes   string `json:"notes,omitempty"`
	}{
		FileURL: req.FileURL,
		Notes:   req.Notes,
	}
	return c.doJSON(ctx, req.Merchant, http.MethodPost, fmt.Sprintf("/api/payin/order/s/%s/dispute/appeal", url.PathEscape(req.OrderID)), body, nil)
}

func (c *ClientImpl) BindPayPalAgreement(ctx context.Context, req *BindPayPalAgreementReq) (*BindPayPalAgreementResp, error) {
	if req == nil {
		return nil, errors.New("payment: bind paypal agreement request is required")
	}
	if err := req.Valid(); err != nil {
		return nil, err
	}

	body := struct {
		Currency CurrencyTp `json:"currency"`
		AppName  string     `json:"app_name"`
		UserID   string     `json:"user_id"`
	}{
		Currency: req.Currency,
		AppName:  req.AppName,
		UserID:   req.UserID,
	}

	var out BindPayPalAgreementResp
	if err := c.doJSON(ctx, req.Merchant, http.MethodPost, "/api/payin/order/agreement", body, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

func (c *ClientImpl) CancelPayPalAgreement(ctx context.Context, req *CancelPayPalAgreementReq) error {
	if req == nil {
		return errors.New("payment: cancel paypal agreement request is required")
	}
	if err := req.Valid(); err != nil {
		return err
	}

	body := struct {
		PayPalEmail string `json:"paypalEmail"`
		AppName     string `json:"app_name"`
		PlayerID    string `json:"player_id"`
	}{
		PayPalEmail: req.PayPalEmail,
		AppName:     req.AppName,
		PlayerID:    req.UserID,
	}
	return c.doJSON(ctx, req.Merchant, http.MethodPost, "/api/payin/order/agreement/cancel", body, nil)
}

func (c *ClientImpl) ApprovePayPalAgreement(ctx context.Context, req *ApprovePayPalAgreementReq) error {
	if req == nil {
		return errors.New("payment: approve paypal agreement request is required")
	}
	if err := req.Valid(); err != nil {
		return err
	}

	body := struct {
		Token string `json:"token"`
	}{
		Token: req.Token,
	}
	return c.doJSON(ctx, req.Merchant, http.MethodPost, "/api/payin/order/agreement/approve", body, nil)
}

func (c *ClientImpl) CreatePayout(ctx context.Context, req *CreatePayoutReq) (*CreatePayoutResp, error) {
	if err := req.Valid(); err != nil {
		return nil, err
	}

	var out CreatePayoutResp
	if err := c.doJSON(ctx, req.Merchant, http.MethodPost, "/api/payout/order", req, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

func (c *ClientImpl) GetPayout(ctx context.Context, req *GetPayoutReq) (*PayoutOrderResp, error) {
	if err := req.Valid(); err != nil {
		return nil, err
	}

	var out PayoutOrderResp
	if err := c.doJSON(ctx, req.Merchant, http.MethodGet, fmt.Sprintf("/api/payout/order/s/%s", url.PathEscape(req.OrderID)), nil, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

func (c *ClientImpl) doJSON(ctx context.Context, merchant Merchant, method string, path string, body any, out any) error {
	merchantID := merchant.ID
	if strings.TrimSpace(merchantID) == "" {
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

	timestamp := fmt.Sprintf("%d", time.Now().Unix())
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

	resp, err := c.httpClient.Do(req)
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

func (c *ClientImpl) GetNotifyIdentity(notify *Notify) (*NotifyIdentity, error) {
	if notify == nil {
		return nil, errors.New("payment: notify is required")
	}
	return notify.Identity()
}

func (c *ClientImpl) ParseNotify(ctx context.Context, secret string, notify *Notify) (*NotifyResp, error) {
	if notify == nil {
		return nil, errors.New("payment: notify is required")
	}
	req, err := notify.ToNotifyReq(secret)
	if err != nil {
		return nil, err
	}
	if ok := req.Verify(); !ok {
		return nil, errors.New("payment: verify notify request failed")
	}
	resp, err := req.ToNotifyResp()
	if err != nil {
		return nil, err
	}
	if err := req.ValidatePayloadOrderID(resp); err != nil {
		return nil, err
	}
	return resp, nil
}

func (c *ClientImpl) NotifySuccess(w http.ResponseWriter) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte(`{"status":"success"}`))
}

func (c *ClientImpl) NotifyFailed(w http.ResponseWriter) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusInternalServerError)
	_, _ = w.Write([]byte(`{"status":"failed"}`))
}
