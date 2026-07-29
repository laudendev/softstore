package cartsession

import (
	"crypto/rand"
	"encoding/base64"
	"net/http"
	"time"
)

const cookieName = "softstore_cart"
const cookieDuration = 30 * 24 * time.Hour

// SecureCookies controls whether the cart cookie requires HTTPS.
// Set to true in production; false for local HTTP-only development.
var SecureCookies = true

// Token returns the cart token from the request's cookie, generating and
// setting a new one via w if none exists yet. Handlers should call this
// once per request rather than reading the cookie directly.
func Token(w http.ResponseWriter, r *http.Request) string {
	if cookie, err := r.Cookie(cookieName); err == nil && cookie.Value != "" {
		return cookie.Value
	}

	token := randomToken()
	http.SetCookie(w, &http.Cookie{
		Name:     cookieName,
		Value:    token,
		Path:     "/",
		HttpOnly: true,
		Secure:   SecureCookies,
		SameSite: http.SameSiteLaxMode,
		Expires:  time.Now().Add(cookieDuration),
	})
	return token
}

func randomToken() string {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		panic(err)
	}
	return base64.URLEncoding.EncodeToString(b)
}
