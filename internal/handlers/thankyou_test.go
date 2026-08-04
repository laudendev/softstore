package handlers

import (
	"html/template"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"softstore/web"
)

func newTestThankYouTmpl(t *testing.T) *template.Template {
	t.Helper()
	tmpl, err := template.ParseFS(web.Templates,
		"templates/layout.html",
		"templates/thank_you.html",
		"templates/session_status_fragment.html",
	)
	if err != nil {
		t.Fatalf("failed to parse thank-you templates: %v", err)
	}
	return tmpl
}

func TestThankYouRendersLoadingStateWithSessionID(t *testing.T) {
	conn := newTestDBWithSchema(t)
	tmpl := newTestThankYouTmpl(t)

	req := httptest.NewRequest(http.MethodGet, "/thank-you?session_id=cs_test_123", nil)
	w := httptest.NewRecorder()

	ThankYou(conn, tmpl)(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	body := w.Body.String()
	if !strings.Contains(body, `hx-get="/session-status/cs_test_123"`) {
		t.Errorf("expected polling hx-get for the session id, got body: %s", body)
	}
	if !strings.Contains(body, "Signing your license key") {
		t.Errorf("expected loading text, got body: %s", body)
	}
}

func TestThankYouRendersFallbackWithoutSessionID(t *testing.T) {
	conn := newTestDBWithSchema(t)
	tmpl := newTestThankYouTmpl(t)

	req := httptest.NewRequest(http.MethodGet, "/thank-you", nil)
	w := httptest.NewRecorder()

	ThankYou(conn, tmpl)(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	body := w.Body.String()
	if !strings.Contains(body, "Check your email") {
		t.Errorf("expected fallback message when no session_id, got body: %s", body)
	}
}
