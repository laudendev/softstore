package handlers

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"softstore/internal/auth"
)

func TestRequireAdminBlocksUnauthenticated(t *testing.T) {
	secret := []byte("test-secret-32-bytes-long-enough")
	called := false

	handler := RequireAdmin(secret, func(w http.ResponseWriter, r *http.Request) {
		called = true
		w.WriteHeader(http.StatusOK)
	})

	req := httptest.NewRequest(http.MethodGet, "/admin/products/new", nil)
	w := httptest.NewRecorder()

	handler(w, req)

	if called {
		t.Error("expected wrapped handler NOT to be called for unauthenticated request")
	}
	if w.Code != http.StatusSeeOther {
		t.Errorf("expected redirect status %d, got %d", http.StatusSeeOther, w.Code)
	}
	if loc := w.Header().Get("Location"); loc != "/admin/login" {
		t.Errorf("expected redirect to /admin/login, got %q", loc)
	}
}

func TestRequireAdminAllowsAuthenticated(t *testing.T) {
	secret := []byte("test-secret-32-bytes-long-enough")
	called := false

	handler := RequireAdmin(secret, func(w http.ResponseWriter, r *http.Request) {
		called = true
		w.WriteHeader(http.StatusOK)
	})

	// Set a valid session cookie the same way the real login flow does.
	setter := httptest.NewRecorder()
	auth.SetSessionCookie(setter, secret)
	sessionCookie := setter.Result().Cookies()[0]

	req := httptest.NewRequest(http.MethodGet, "/admin/products/new", nil)
	req.AddCookie(sessionCookie)
	w := httptest.NewRecorder()

	handler(w, req)

	if !called {
		t.Error("expected wrapped handler to be called for authenticated request")
	}
	if w.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, w.Code)
	}
}

func TestRequireInternalSecretBlocksMissingHeader(t *testing.T) {
	called := false

	handler := RequireInternalSecret("correct-secret", func(w http.ResponseWriter, r *http.Request) {
		called = true
		w.WriteHeader(http.StatusOK)
	})

	req := httptest.NewRequest(http.MethodGet, "/internal/products/by-price/price_x", nil)
	w := httptest.NewRecorder()

	handler(w, req)

	if called {
		t.Error("expected wrapped handler NOT to be called without the secret header")
	}
	if w.Code != http.StatusUnauthorized {
		t.Errorf("expected status %d, got %d", http.StatusUnauthorized, w.Code)
	}
}

func TestRequireInternalSecretBlocksWrongSecret(t *testing.T) {
	called := false

	handler := RequireInternalSecret("correct-secret", func(w http.ResponseWriter, r *http.Request) {
		called = true
		w.WriteHeader(http.StatusOK)
	})

	req := httptest.NewRequest(http.MethodGet, "/internal/products/by-price/price_x", nil)
	req.Header.Set("X-Internal-Secret", "wrong-secret")
	w := httptest.NewRecorder()

	handler(w, req)

	if called {
		t.Error("expected wrapped handler NOT to be called with wrong secret")
	}
	if w.Code != http.StatusUnauthorized {
		t.Errorf("expected status %d, got %d", http.StatusUnauthorized, w.Code)
	}
}

func TestRequireInternalSecretAllowsCorrectSecret(t *testing.T) {
	called := false

	handler := RequireInternalSecret("correct-secret", func(w http.ResponseWriter, r *http.Request) {
		called = true
		w.WriteHeader(http.StatusOK)
	})

	req := httptest.NewRequest(http.MethodGet, "/internal/products/by-price/price_x", nil)
	req.Header.Set("X-Internal-Secret", "correct-secret")
	w := httptest.NewRecorder()

	handler(w, req)

	if !called {
		t.Error("expected wrapped handler to be called with correct secret")
	}
	if w.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, w.Code)
	}
}
