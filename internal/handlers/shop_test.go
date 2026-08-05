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

func TestShopEmptyCatalog(t *testing.T) {
	conn := newTestDBWithSchema(t)
	tmpl := testTemplate(t)

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	w := httptest.NewRecorder()

	Shop(conn, tmpl)(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", w.Code)
	}
	if !strings.Contains(w.Body.String(), "No products yet.") {
		t.Error("expected empty-catalog message in response body")
	}
}

func TestShopWithProducts(t *testing.T) {
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

	Shop(conn, tmpl)(w, req)

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

func TestCartCountForRequestNoCookie(t *testing.T) {
	conn := newTestDBWithSchema(t)

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	count := cartCountForRequest(conn, req)

	if count != 0 {
		t.Errorf("expected 0 for request with no cart cookie, got %d", count)
	}
}

func TestCartCountForRequestExistingCart(t *testing.T) {
	conn := newTestDBWithSchema(t)

	p := &models.Product{
		Name: "Test Widget", Slug: "test-widget", PriceCents: 1999,
		StripePriceID: "price_abc123", ProductCode: "TWDG", TaxCode: "txcd_10202000",
	}
	if err := db.CreateProduct(conn, p); err != nil {
		t.Fatalf("failed to seed product: %v", err)
	}

	cart, err := db.GetOrCreateCart(conn, "existing-token")
	if err != nil {
		t.Fatalf("get or create cart: %v", err)
	}
	if err := db.AddCartItem(conn, cart.ID, p.ID, 1, 3); err != nil {
		t.Fatalf("add cart item: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.AddCookie(&http.Cookie{Name: "softstore_cart", Value: "existing-token"})
	count := cartCountForRequest(conn, req)

	if count != 3 {
		t.Errorf("expected 3, got %d", count)
	}
}
