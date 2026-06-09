package payment

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"strings"
)

type VerifyNotifyRequest struct {
	Secret    string
	Timestamp string
	Nonce     string
	Signature string
	Body      []byte
}

type NotifyEvent struct {
	NotifyType string         `json:"notify_type"`
	Raw        map[string]any `json:"-"`
	body       []byte
}

func VerifyNotify(req VerifyNotifyRequest) bool {
	expected := signNotify(req.Secret, req.Timestamp, req.Nonce, req.Body)
	return hmac.Equal([]byte(strings.TrimSpace(req.Signature)), []byte(expected))
}

func VerifyNotifyHTTP(secret string, header http.Header, body []byte) bool {
	return VerifyNotify(VerifyNotifyRequest{
		Secret:    secret,
		Timestamp: header.Get(headerTimestamp),
		Nonce:     header.Get(headerNonce),
		Signature: header.Get(headerSignature),
		Body:      body,
	})
}

func ParseNotify(body []byte) (*NotifyEvent, error) {
	var raw map[string]any
	if err := json.Unmarshal(body, &raw); err != nil {
		return nil, err
	}
	notifyType, _ := raw["notify_type"].(string)
	return &NotifyEvent{
		NotifyType: strings.TrimSpace(notifyType),
		Raw:        raw,
		body:       append([]byte(nil), body...),
	}, nil
}

func (e *NotifyEvent) Decode(v any) error {
	return json.Unmarshal(e.body, v)
}

func WriteNotifyAck(w http.ResponseWriter) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte(`{"status":"success"}`))
}

func signNotify(secret, timestamp, nonce string, body []byte) string {
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte(timestamp))
	mac.Write([]byte("\n"))
	mac.Write([]byte(nonce))
	mac.Write([]byte("\n"))
	mac.Write(body)
	return hex.EncodeToString(mac.Sum(nil))
}
