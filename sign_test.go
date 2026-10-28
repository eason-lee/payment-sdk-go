package payment

import "testing"

func TestCanonicalQuery(t *testing.T) {
	cases := []struct {
		in, want string
	}{
		{"", ""},
		{"b=2&a=1", "a=1&b=2"},
		{"a=2&a=1", "a=1&a=2"},
		{"  ", ""},
	}
	for _, tc := range cases {
		if got := CanonicalQuery(tc.in); got != tc.want {
			t.Fatalf("CanonicalQuery(%q)=%q want %q", tc.in, got, tc.want)
		}
	}
}

func TestCanonicalStringStableMatchesPaypaya(t *testing.T) {
	got := CanonicalString("POST", "/api/x", "b=2&a=1", "1710000000", "n1", "100", []byte("hi"))
	want := "PAYPAYA-HMAC-SHA256\nPOST\n/api/x\na=1&b=2\n1710000000\nn1\n100\n8f434346648f6b96df89dda901c5176b10a6d83961dd3c1ac88b59b2dc327aa4"
	if got != want {
		t.Fatalf("got:\n%s\nwant:\n%s", got, want)
	}
}
