package handlers

import (
	"html/template"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"softstore/web"
)

func TestLegalPageRendersTitleAndContent(t *testing.T) {
	conn := newTestDBWithSchema(t)

	tmpl, err := template.ParseFS(web.Templates,
		"templates/layout.html",
		"templates/legal_layout.html",
		"templates/legal_terms.html",
	)
	if err != nil {
		t.Fatalf("failed to parse legal terms template: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/legal/terms", nil)
	w := httptest.NewRecorder()

	LegalPage(conn, tmpl, "Terms of Service")(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	body := w.Body.String()
	if !strings.Contains(body, "Terms of Service") {
		t.Errorf("expected page title in body, got: %s", body)
	}
	if !strings.Contains(body, "lauden.dev") {
		t.Errorf("expected legal content to be present, got: %s", body)
	}
}
