package handlers

import (
	"database/sql"
	"html/template"
	"log"
	"net/http"

	"softstore/internal/quartermaster"
)

// ThankYouData is the data passed to the thank-you page template.
type ThankYouData struct {
	Title     string
	CartCount int64
	SessionID string
	ShowCart bool
}

// ThankYou renders the post-checkout thank-you page immediately, with a
// loading state. The page polls SessionStatus via HTMX to reveal the
// receipt once fulfillment completes.
func ThankYou(conn *sql.DB, tmpl *template.Template) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		data := ThankYouData{
			Title:     "Thank You",
			CartCount: cartCountForRequest(conn, r),
			SessionID: r.URL.Query().Get("session_id"),
		}
		if err := tmpl.ExecuteTemplate(w, "layout", data); err != nil {
			log.Println("render thank you:", err)
			http.Error(w, "internal error", http.StatusInternalServerError)
		}
	}
}

// SessionStatusData is the data passed to the session-status fragment
// template, rendered on each poll.
type SessionStatusData struct {
	Ready     bool
	Items     []quartermaster.ReceiptItem
	TaxLine   string
	TotalLine string
}

// SessionStatus handles GET /session-status/{session_id}. It polls
// Quartermaster for fulfillment status and renders either the loading
// fragment (still processing) or the receipt fragment (ready), for
// HTMX to swap into the thank-you page.
func SessionStatus(client *quartermaster.Client, tmpl *template.Template) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		sessionID := r.PathValue("session_id")
		if sessionID == "" {
			http.Error(w, "bad request", http.StatusBadRequest)
			return
		}

		status, err := client.GetSessionStatus(sessionID)
		if err != nil {
			log.Println("session status poll failed for", sessionID, ":", err)
			// Render the loading fragment again rather than erroring out —
			// a transient network hiccup shouldn't break the polling UI;
			// the next poll will likely succeed.
			status = quartermaster.SessionStatus{Ready: false}
		}

		data := SessionStatusData{
			Ready:     status.Ready,
			Items:     status.Items,
			TaxLine:   status.TaxLine,
			TotalLine: status.TotalLine,
		}
		
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		if err := tmpl.ExecuteTemplate(w, "session-status-fragment", data); err != nil {
			log.Println("render session status fragment:", err)
			http.Error(w, "internal error", http.StatusInternalServerError)
		}
	}
}
