package payment

import (
	"crypto/ed25519"
	"crypto/sha256"
	"crypto/x509"
	"encoding/base64"
	"encoding/hex"
	"encoding/pem"
	"errors"
	"fmt"
	"strconv"
	"strings"
)

// CanonicalCallbackString builds the CallbackV2 signed payload (LF newlines).
//
//	PAYPAYA-CALLBACK-ED25519
//	TS
//	NONCE
//	NOTIFY_TP
//	ORDER_ID
//	hex(sha256(body))
func CanonicalCallbackString(timestamp, nonce string, notifyTp int, orderID string, body []byte) string {
	sum := sha256.Sum256(body)
	var b strings.Builder
	b.WriteString(callbackSignPrefixV2)
	b.WriteByte('\n')
	b.WriteString(timestamp)
	b.WriteByte('\n')
	b.WriteString(nonce)
	b.WriteByte('\n')
	b.WriteString(strconv.Itoa(notifyTp))
	b.WriteByte('\n')
	b.WriteString(orderID)
	b.WriteByte('\n')
	b.WriteString(hex.EncodeToString(sum[:]))
	return b.String()
}

// ParseCallbackPublicKey accepts hex, std/raw base64 of 32-byte pubkey, or PKIX/SPKI PEM.
func ParseCallbackPublicKey(material string) (ed25519.PublicKey, error) {
	material = strings.TrimSpace(material)
	if material == "" {
		return nil, errors.New("payment: callback public key is required")
	}
	if strings.Contains(material, "BEGIN") {
		block, _ := pem.Decode([]byte(material))
		if block == nil {
			return nil, errors.New("payment: callback public key pem decode failed")
		}
		parsed, err := x509.ParsePKIXPublicKey(block.Bytes)
		if err != nil {
			return nil, fmt.Errorf("payment: callback public key pkix: %w", err)
		}
		pub, ok := parsed.(ed25519.PublicKey)
		if !ok {
			return nil, fmt.Errorf("payment: callback public key type %T not ed25519", parsed)
		}
		out := make(ed25519.PublicKey, len(pub))
		copy(out, pub)
		return out, nil
	}
	var raw []byte
	var err error
	if raw, err = hex.DecodeString(material); err != nil {
		if raw, err = base64.StdEncoding.DecodeString(material); err != nil {
			if raw, err = base64.RawStdEncoding.DecodeString(material); err != nil {
				return nil, errors.New("payment: callback public key: expected hex, base64, or PEM")
			}
		}
	}
	if len(raw) != ed25519.PublicKeySize {
		return nil, fmt.Errorf("payment: callback public key must be %d bytes", ed25519.PublicKeySize)
	}
	out := make(ed25519.PublicKey, ed25519.PublicKeySize)
	copy(out, raw)
	return out, nil
}

// VerifyCallbackV2 checks base64(Ed25519) over CanonicalCallbackString.
func VerifyCallbackV2(public ed25519.PublicKey, timestamp, nonce string, notifyTp int, orderID string, body []byte, sigB64 string) error {
	if len(public) != ed25519.PublicKeySize {
		return errors.New("payment: callback public key invalid")
	}
	sig, err := base64.StdEncoding.DecodeString(strings.TrimSpace(sigB64))
	if err != nil {
		return fmt.Errorf("payment: callback signature base64: %w", err)
	}
	msg := CanonicalCallbackString(timestamp, nonce, notifyTp, orderID, body)
	if !ed25519.Verify(public, []byte(msg), sig) {
		return errors.New("payment: callback signature invalid")
	}
	return nil
}

// SignCallbackV2ForTest signs with a private key; for unit tests only.
func SignCallbackV2ForTest(private ed25519.PrivateKey, timestamp, nonce string, notifyTp int, orderID string, body []byte) (string, error) {
	if len(private) != ed25519.PrivateKeySize {
		return "", errors.New("payment: callback private key invalid")
	}
	sig := ed25519.Sign(private, []byte(CanonicalCallbackString(timestamp, nonce, notifyTp, orderID, body)))
	return base64.StdEncoding.EncodeToString(sig), nil
}
