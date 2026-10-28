package payment

import (
	"crypto/ed25519"
	"crypto/rand"
	"encoding/base64"
	"testing"
)

func TestCallbackV2RoundTrip(t *testing.T) {
	pub, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	body := []byte(`{"order_id":"1"}`)
	ts, nonce := "1710000000", "nonce-1"
	sig, err := SignCallbackV2ForTest(priv, ts, nonce, 1, "200001", body)
	if err != nil {
		t.Fatal(err)
	}
	if err := VerifyCallbackV2(pub, ts, nonce, 1, "200001", body, sig); err != nil {
		t.Fatal(err)
	}
	if err := VerifyCallbackV2(pub, ts, nonce, 1, "200001", []byte(`{}`), sig); err == nil {
		t.Fatal("expected verify fail on body change")
	}
	parsed, err := ParseCallbackPublicKey(base64.StdEncoding.EncodeToString(pub))
	if err != nil {
		t.Fatal(err)
	}
	if err := VerifyCallbackV2(parsed, ts, nonce, 1, "200001", body, sig); err != nil {
		t.Fatal(err)
	}
}

func TestCanonicalCallbackStringStable(t *testing.T) {
	got := CanonicalCallbackString("1", "n", 2, "oid", []byte("abc"))
	wantPrefix := "PAYPAYA-CALLBACK-ED25519\n1\nn\n2\noid\n"
	if got[:len(wantPrefix)] != wantPrefix {
		t.Fatalf("got %q", got)
	}
}
