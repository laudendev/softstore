package auth

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"golang.org/x/crypto/bcrypt"
)

func TestCheckPassword(t *testing.T) {
	hash, err := bcrypt.GenerateFromPassword([]byte("correct-horse-battery-staple"), bcrypt.DefaultCost)
	if err != nil {
		t.Fatalf("failed to generate test hash: %v", err)
	}

	if !CheckPassword(string(hash), "correct-horse-battery-staple") {
		t.Error("expected correct password to pass")
	}
	if CheckPassword(string(hash), "wrong-password") {
		t.Error("expected wrong password to fail")
	}
	if CheckPassword(string(hash), "") {
		t.Error("expected empty password to fail")
	}
}

func TestSignAndVerifyToken(t *testing.T) {
	secret := []byte("test-secret-32-bytes-long-enough")
	expiry := time.Now().Add(1 * time.Hour).Unix()

	token := signToken(secret, expiry)
	if !verifyToken(secret, token) {
		t.Error("expected freshly signed token to verify")
	}
}

func TestVerifyTokenExpired(t *testing.T) {
	secret := []byte("test-secret-32-bytes-long-enough")
	expiry := time.Now().Add(-1 * time.Hour).Unix() // already expired

	token := signToken(secret, expiry)
	if verifyToken(secret, token) {
		t.Error("expected expired token to fail verification")
	}
}

func TestVerifyTokenWrongSecret(t *testing.T) {
	secret := []byte("test-secret-32-bytes-long-enough")
	wrongSecret := []byte("different-secret-entirely-here!")
	expiry := time.Now().Add(1 * time.Hour).Unix()

	token := signToken(secret, expiry)
	if verifyToken(wrongSecret, token) {
		t.Error("expected token signed with different secret to fail verification")
	}
}

func TestVerifyTokenTampered(t *testing.T) {
	secret := []byte("test-secret-32-bytes-long-enough")
	expiry := time.Now().Add(1 * time.Hour).Unix()

	token := signToken(secret, expiry)
	tampered := token[:len(token)-1] + "X" // flip the last character of the signature

	if verifyToken(secret, tampered) {
		t.Error("expected tampered token to fail verification")
	}
}

func TestVerifyTokenMalformed(t *testing.T) {
	secret := []byte("test-secret-32-bytes-long-enough")

	cases := []string{"", "no-dot-separator", ".", "..", "short"}
	for _, c := range cases {
		if verifyToken(secret, c) {
			t.Errorf("expected malformed token %q to fail verification", c)
		}
	}
}

func TestSetAndClearSessionCookie(t *testing.T) {
	secret := []byte("test-secret-32-bytes-long-enough")

	w := httptest.NewRecorder()
	SetSessionCookie(w, secret)

	resp := w.Result()
	cookies := resp.Cookies()
	if len(cookies) != 1 {
		t.Fatalf("expected 1 cookie to be set, got %d", len(cookies))
	}
	if cookies[0].Name != sessionCookieName {
		t.Errorf("expected cookie name %q, got %q", sessionCookieName, cookies[0].Name)
	}
	if !cookies[0].HttpOnly {
		t.Error("expected session cookie to be HttpOnly")
	}

	// Simulate a request carrying this cookie, confirm IsAuthenticated sees it as valid.
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.AddCookie(cookies[0])
	if !IsAuthenticated(req, secret) {
		t.Error("expected request with valid session cookie to be authenticated")
	}

	// Now clear it, and confirm the cleared cookie has an expiry in the past.
	w2 := httptest.NewRecorder()
	ClearSessionCookie(w2)
	cleared := w2.Result().Cookies()[0]
	if cleared.MaxAge >= 0 {
		t.Error("expected cleared cookie to have negative MaxAge (immediate expiry)")
	}
}

func TestIsAuthenticatedNoCookie(t *testing.T) {
	secret := []byte("test-secret-32-bytes-long-enough")
	req := httptest.NewRequest(http.MethodGet, "/", nil)

	if IsAuthenticated(req, secret) {
		t.Error("expected request with no cookie to be unauthenticated")
	}
}
