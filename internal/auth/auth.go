package auth

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"net/http"
	"time"

	"golang.org/x/crypto/bcrypt"
)

const sessionCookieName = "softstore_admin_session"
const sessionDuration = 24 * time.Hour

// CheckPassword compares a plaintext password against a stored bcrypt hash.
func CheckPassword(hash, password string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
	return err == nil
}

// signToken creates an HMAC-signed value: "expiry.signature"
func signToken(secret []byte, expiry int64) string {
	msg := encodeInt64(expiry)
	mac := hmac.New(sha256.New, secret)
	mac.Write([]byte(msg))
	sig := base64.URLEncoding.EncodeToString(mac.Sum(nil))
	return msg + "." + sig
}

// verifyToken checks the signature and expiry on a session token.
func verifyToken(secret []byte, token string) bool {
	if len(token) < 20 {
		return false
	}
	sepIdx := -1
	for i := len(token) - 1; i >= 0; i-- {
		if token[i] == '.' {
			sepIdx = i
			break
		}
	}
	if sepIdx == -1 {
		return false
	}
	msg, sig := token[:sepIdx], token[sepIdx+1:]

	mac := hmac.New(sha256.New, secret)
	mac.Write([]byte(msg))
	expectedSig := base64.URLEncoding.EncodeToString(mac.Sum(nil))
	if !hmac.Equal([]byte(sig), []byte(expectedSig)) {
		return false
	}

	expiry := decodeInt64(msg)
	return time.Now().Unix() < expiry
}

func encodeInt64(v int64) string {
	return base64.URLEncoding.EncodeToString([]byte(time.Unix(v, 0).UTC().Format(time.RFC3339)))
}

func decodeInt64(s string) int64 {
	b, err := base64.URLEncoding.DecodeString(s)
	if err != nil {
		return 0
	}
	t, err := time.Parse(time.RFC3339, string(b))
	if err != nil {
		return 0
	}
	return t.Unix()
}

// SetSessionCookie issues a signed, expiring session cookie.
func SetSessionCookie(w http.ResponseWriter, secret []byte) {
	expiry := time.Now().Add(sessionDuration).Unix()
	token := signToken(secret, expiry)
	http.SetCookie(w, &http.Cookie{
		Name:     sessionCookieName,
		Value:    token,
		Path:     "/",
		HttpOnly: true,
		Secure:   true,
		SameSite: http.SameSiteStrictMode,
		Expires:  time.Now().Add(sessionDuration),
	})
}

// ClearSessionCookie logs the admin out.
func ClearSessionCookie(w http.ResponseWriter) {
	http.SetCookie(w, &http.Cookie{
		Name:     sessionCookieName,
		Value:    "",
		Path:     "/",
		HttpOnly: true,
		Secure:   true,
		SameSite: http.SameSiteStrictMode,
		MaxAge:   -1,
	})
}

// IsAuthenticated checks the request's session cookie.
func IsAuthenticated(r *http.Request, secret []byte) bool {
	cookie, err := r.Cookie(sessionCookieName)
	if err != nil {
		return false
	}
	return verifyToken(secret, cookie.Value)
}

// RandomSecret generates a cryptographically secure random secret, useful for local dev.
func RandomSecret() []byte {
	b := make([]byte, 32)
	_, err := rand.Read(b)
	if err != nil {
		panic(err)
	}
	return b
}
