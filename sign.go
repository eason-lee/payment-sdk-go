package payment

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"net/url"
	"sort"
	"strings"
)

// CanonicalQuery sorts query pairs as key=value joined by &; empty if none.
// Matches paypaya merchant.CanonicalQuery (ParseQuery, sort keys, sort values per key).
func CanonicalQuery(rawQuery string) string {
	rawQuery = strings.TrimSpace(rawQuery)
	if rawQuery == "" {
		return ""
	}
	values, err := url.ParseQuery(rawQuery)
	if err != nil {
		return ""
	}
	keys := make([]string, 0, len(values))
	for k := range values {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	parts := make([]string, 0, len(values))
	for _, k := range keys {
		vs := append([]string(nil), values[k]...)
		sort.Strings(vs)
		for _, v := range vs {
			parts = append(parts, k+"="+v)
		}
	}
	return strings.Join(parts, "&")
}

// CanonicalString builds the HMAC-SHA256 v2 signing string (LF newlines).
//
//	PAYPAYA-HMAC-SHA256
//	METHOD
//	PATH
//	canonical-query
//	TS
//	NONCE
//	MERCHANT_ID
//	hex(sha256(body))
func CanonicalString(method, path, rawQuery, timestamp, nonce, merchantID string, body []byte) string {
	sum := sha256.Sum256(body)
	var b strings.Builder
	b.WriteString(signPrefixHMACV2)
	b.WriteByte('\n')
	b.WriteString(method)
	b.WriteByte('\n')
	b.WriteString(path)
	b.WriteByte('\n')
	b.WriteString(CanonicalQuery(rawQuery))
	b.WriteByte('\n')
	b.WriteString(timestamp)
	b.WriteByte('\n')
	b.WriteString(nonce)
	b.WriteByte('\n')
	b.WriteString(merchantID)
	b.WriteByte('\n')
	b.WriteString(hex.EncodeToString(sum[:]))
	return b.String()
}

// SignHMACV2 returns lowercase hex(HMAC-SHA256(secret, canonical v2 string)).
func SignHMACV2(secret, method, path, rawQuery, timestamp, nonce, merchantID string, body []byte) string {
	mac := hmac.New(sha256.New, []byte(secret))
	_, _ = mac.Write([]byte(CanonicalString(method, path, rawQuery, timestamp, nonce, merchantID, body)))
	return hex.EncodeToString(mac.Sum(nil))
}

func requestSignPath(u *url.URL) string {
	if u == nil {
		return ""
	}
	path := u.EscapedPath()
	if path == "" {
		return u.Path
	}
	return path
}
