package handlers

import (
	"crypto/subtle"
	"net/http"

	"softstore/internal/auth"
)

func RequireAdmin(sessionSecret []byte, next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if !auth.IsAuthenticated(r, sessionSecret) {
			http.Redirect(w, r, "/admin/login", http.StatusSeeOther)
			return
		}
		next(w, r)
	}
}


// RequireInternalSecret protects service-to-service endpoints (e.g. those
// called by Quartermaster) with a static shared secret, sent as the
// X-Internal-Secret header, compared in constant time.
func RequireInternalSecret(secret string, next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		provided := r.Header.Get("X-Internal-Secret")
		if subtle.ConstantTimeCompare([]byte(provided), []byte(secret)) != 1 {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}
		next(w, r)
	}
}
