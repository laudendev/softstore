package handlers

import (
	"database/sql"
	"html/template"
	"log"
	"net/http"
)

// LegalPage renders a static legal document (Terms, Privacy, EULA,
// Refund Policy, Cookie Policy) using the shared layout. Each page has
// its own pre-parsed *template.Template (a clone of a shared base with
// only that page's content block parsed in), so title is the only
// per-route data needed.
func LegalPage(conn *sql.DB, tmpl *template.Template, title string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		data := ShopData{
			Title:     title,
			CartCount: cartCountForRequest(conn, r),
		}
		if err := tmpl.ExecuteTemplate(w, "layout", data); err != nil {
			log.Println("render legal page:", err)
			http.Error(w, "internal error", http.StatusInternalServerError)
		}
	}
}
