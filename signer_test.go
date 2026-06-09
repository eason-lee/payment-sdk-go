package payment

import "testing"

func TestSignRequest(t *testing.T) {
	got := SignRequest("secret", SignRequestInput{
		Method:    "POST",
		Path:      "/api/payin/order",
		Timestamp: "1700000000",
		Nonce:     "nonce-abc",
		Body:      []byte(`{"amount":100}`),
	})
	want := "71c214883569b3f91f4d2514cba34af8763942c0d4b98f7b645e9acb673322ed"
	if got != want {
		t.Fatalf("signature mismatch\nwant: %s\n got: %s", want, got)
	}
}

func TestSignRequestEmptyBody(t *testing.T) {
	got := SignRequest("secret", SignRequestInput{
		Method:    "GET",
		Path:      "/api/payin/order/123",
		Timestamp: "1700000000",
		Nonce:     "nonce-abc",
		Body:      nil,
	})
	want := "d281574a8e2cd83c9b28f9b5fb643148ffc7c6e08250494e896c79f73256cce4"
	if got != want {
		t.Fatalf("signature mismatch\nwant: %s\n got: %s", want, got)
	}
}

func TestSignRequestBodyChangesSignature(t *testing.T) {
	a := SignRequest("secret", SignRequestInput{
		Method:    "POST",
		Path:      "/api/payin/order",
		Timestamp: "1700000000",
		Nonce:     "nonce-abc",
		Body:      []byte(`{"amount":100}`),
	})
	b := SignRequest("secret", SignRequestInput{
		Method:    "POST",
		Path:      "/api/payin/order",
		Timestamp: "1700000000",
		Nonce:     "nonce-abc",
		Body:      []byte(`{"amount":101}`),
	})
	if a == b {
		t.Fatal("expected body change to change signature")
	}
}

func TestGenerateNonce(t *testing.T) {
	nonce, err := GenerateNonce()
	if err != nil {
		t.Fatal(err)
	}
	if len(nonce) != 32 {
		t.Fatalf("nonce length = %d", len(nonce))
	}
}
