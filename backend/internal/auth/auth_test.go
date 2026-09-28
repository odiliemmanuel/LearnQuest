package auth

import "testing"

func TestVerificationCodeHashIsSecretScoped(t *testing.T) {
	service := &Service{OTPSecret: "first-secret"}
	first := service.hashCode(42, "123456")
	if first == "" {
		t.Fatal("expected a code hash")
	}
	if first == (&Service{OTPSecret: "second-secret"}).hashCode(42, "123456") {
		t.Fatal("hash must change when the OTP secret changes")
	}
	if first == service.hashCode(43, "123456") {
		t.Fatal("hash must be bound to the user")
	}
}

func TestGenerateCodeIsSixDigits(t *testing.T) {
	code, err := generateCode()
	if err != nil {
		t.Fatalf("generateCode returned an error: %v", err)
	}
	if len(code) != 6 {
		t.Fatalf("expected six digits, got %q", code)
	}
	for _, char := range code {
		if char < '0' || char > '9' {
			t.Fatalf("expected numeric code, got %q", code)
		}
	}
}
