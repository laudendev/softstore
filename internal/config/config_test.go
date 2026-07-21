package config

import (
	"os"
	"testing"
)

func TestRequireEnvPresent(t *testing.T) {
	t.Setenv("TEST_REQUIRED_VAR", "some-value")

	got := requireEnv("TEST_REQUIRED_VAR")
	if got != "some-value" {
		t.Errorf("expected 'some-value', got %q", got)
	}
}

func TestSecureCookiesTrue(t *testing.T) {
	t.Setenv("SECURE_COOKIES", "true")
	if !SecureCookies() {
		t.Error("expected SecureCookies() to be true when SECURE_COOKIES=true")
	}
}

func TestSecureCookiesFalseByDefault(t *testing.T) {
	os.Unsetenv("SECURE_COOKIES")
	if SecureCookies() {
		t.Error("expected SecureCookies() to be false when unset")
	}
}

func TestSecureCookiesAnyOtherValueIsFalse(t *testing.T) {
	t.Setenv("SECURE_COOKIES", "yes")
	if SecureCookies() {
		t.Error("expected SecureCookies() to be false for any value other than exactly 'true'")
	}
}

func TestStripeSecretKeyReturnsValue(t *testing.T) {
	t.Setenv("STRIPE_SECRET_KEY", "sk_test_abc123")
	got := StripeSecretKey()
	if got != "sk_test_abc123" {
		t.Errorf("expected 'sk_test_abc123', got %q", got)
	}
}

func TestBaseURLReturnsValue(t *testing.T) {
	t.Setenv("BASE_URL", "https://store.example.com")
	got := BaseURL()
	if got != "https://store.example.com" {
		t.Errorf("expected the configured base URL, got %q", got)
	}
}

func TestAdminUsernameReturnsValue(t *testing.T) {
	t.Setenv("ADMIN_USERNAME", "tylerl")
	got := AdminUsername()
	if got != "tylerl" {
		t.Errorf("expected 'tylerl', got %q", got)
	}
}

func TestAdminPasswordHashReturnsValue(t *testing.T) {
	t.Setenv("ADMIN_PASSWORD_HASH", "$2a$10$examplehash")
	got := AdminPasswordHash()
	if got != "$2a$10$examplehash" {
		t.Errorf("expected the configured hash, got %q", got)
	}
}

func TestSessionSecretReturnsBytes(t *testing.T) {
	t.Setenv("SESSION_SECRET", "my-secret-value")
	got := SessionSecret()
	if string(got) != "my-secret-value" {
		t.Errorf("expected 'my-secret-value', got %q", string(got))
	}
}
