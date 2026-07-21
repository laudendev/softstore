package handlers

import (
	"database/sql"
	"html/template"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"softstore/internal/db"
	"softstore/internal/models"

	_ "modernc.org/sqlite"
)

func newTestDBWithSchema(t *testing.T) *sql.DB {
	t.Helper()
	conn, err := db.Open(":memory:")
	if err != nil {
		t.Fatalf("failed to open test db: %v", err)
	}
	t.Cleanup(func() { conn.Close() })
	return conn
}

func testTemplate(t *testing.T) *template.Template {
	t.Helper()
	tmplSrc := `
{{define "layout"}}{{template "content" .}}{{end}}
{{define "content"}}
{{if .Products}}
{{range .Products}}<article>{{.Name}} - ${{.PriceDollars}}</article>{{end}}
{{else}}
<p>No products yet.</p>
{{end}}
{{end}}`
	tmpl, err := template.New("test").Parse(tmplSrc)
	if err != nil {
		t.Fatalf("failed to parse test template: %v", err)
	}
	return tmpl
}

func TestHomeEmptyCatalog(t *testing.T) {
	conn := newTestDBWithSchema(t)
	tmpl := testTemplate(t)

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	w := httptest.NewRecorder()

	Home(conn, tmpl)(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", w.Code)
	}
	if !strings.Contains(w.Body.String(), "No products yet.") {
		t.Error("expected empty-catalog message in response body")
	}
}

func TestHomeWithProducts(t *testing.T) {
	conn := newTestDBWithSchema(t)
	tmpl := testTemplate(t)

	p := &models.Product{
		Name: "Test Widget", Slug: "test-widget", PriceCents: 1999,
		StripePriceID: "price_x", ProductCode: "TWDG", TaxCode: "txcd_10202000",
	}
	if err := db.CreateProduct(conn, p); err != nil {
		t.Fatalf("failed to seed product: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	w := httptest.NewRecorder()

	Home(conn, tmpl)(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", w.Code)
	}
	body := w.Body.String()
	if !strings.Contains(body, "Test Widget") {
		t.Error("expected product name in response body")
	}
	if !strings.Contains(body, "19.99") {
		t.Error("expected formatted price in response body")
	}
}
