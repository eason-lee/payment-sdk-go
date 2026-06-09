package payment

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"strings"
	"time"
)

// SignRequestInput is the canonical merchant request signing input.
type SignRequestInput struct {
	Method    string
	Path      string
	Timestamp string
	Nonce     string
	Body      []byte
}

// SignRequest signs a merchant gateway request.
func SignRequest(secret string, in SignRequestInput) string {
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

// GenerateNonce returns a random 32-character alphanumeric nonce.
func GenerateNonce() (string, error) {
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

func formatUnix(t time.Time) string {
	return strconvFormatInt(t.Unix())
}
