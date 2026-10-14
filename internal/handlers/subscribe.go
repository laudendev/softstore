package handlers

import (
	"database/sql"
	"errors"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"net/mail"
	"strings"

	"softstore/internal/db"
)

// SubscribePageData is the data passed to the confirm/unsubscribe pages.
type SubscribePageData struct {
	Title     string
	CartCount int64
	ShowCart  bool
	Heading   string
	Message   string
}

// SubscribeSignup handles POST /subscribe. Expects form field "email".
// Returns an HTMX-swappable HTML fragment with a success or error message.
func SubscribeSignup(conn *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		email := strings.TrimSpace(strings.ToLower(r.FormValue("email")))
		if email == "" {
			writeSubscribeMsg(w, "Please enter an email address.", true)
			return
		}
		if _, err := mail.ParseAddress(email); err != nil {
			writeSubscribeMsg(w, "That doesn't look like a valid email.", true)
			return
		}

		source := "shop"
		if r.FormValue("source") == "popfocus" {
			source = "popfocus"
		}

		confirmToken, err := db.CreateSubscriber(conn, email, source)
		if err != nil {
			if errors.Is(err, db.ErrAlreadySubscribed) {
				writeSubscribeMsg(w, "You're already subscribed.", false)
				return
			}
			log.Println("subscribe signup, create subscriber:", err)
			writeSubscribeMsg(w, "Something went wrong. Try again.", true)
			return
		}

		if err := sendConfirmEmail(email, confirmToken, source); err != nil {
			log.Println("subscribe signup, send confirm email:", err)
			writeSubscribeMsg(w, "Something went wrong sending the confirmation email. Try again.", true)
			return
		}

		writeSubscribeMsg(w, "Check your email to confirm your subscription.", false)
	}
}

func writeSubscribeMsg(w http.ResponseWriter, msg string, isError bool) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	class := "subscribe-success"
	if isError {
		class = "subscribe-error"
	}
	fmt.Fprintf(w, `<p class="%s">%s</p>`, class, msg)
}

// SubscribeConfirm handles GET /subscribe/confirm?token=... — the link
// clicked from the confirmation email.
func SubscribeConfirm(conn *sql.DB, tmpl *template.Template) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		token := r.URL.Query().Get("token")
		if token == "" {
			http.NotFound(w, r)
			return
		}

		if err := db.ConfirmSubscriber(conn, token); err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				http.Error(w, "Invalid or already-used confirmation link.", http.StatusBadRequest)
				return
			}
			log.Println("subscribe confirm:", err)
			http.Error(w, "internal error", http.StatusInternalServerError)
			return
		}

		data := SubscribePageData{
			Title:     "Subscription Confirmed",
			CartCount: cartCountForRequest(conn, r),
			Heading:   "Subscription confirmed",
			Message:   "You're all set. Thanks for signing up.",
		}
		if err := tmpl.ExecuteTemplate(w, "layout", data); err != nil {
			log.Println("render subscribe confirm:", err)
			http.Error(w, "internal error", http.StatusInternalServerError)
		}
	}
}

// SubscribeUnsubscribe handles GET /subscribe/unsubscribe?token=...
func SubscribeUnsubscribe(conn *sql.DB, tmpl *template.Template) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		token := r.URL.Query().Get("token")
		if token == "" {
			http.NotFound(w, r)
			return
		}

		if err := db.UnsubscribeByToken(conn, token); err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				http.Error(w, "Invalid unsubscribe link.", http.StatusBadRequest)
				return
			}
			log.Println("subscribe unsubscribe:", err)
			http.Error(w, "internal error", http.StatusInternalServerError)
			return
		}

		data := SubscribePageData{
			Title:     "Unsubscribed",
			CartCount: cartCountForRequest(conn, r),
			Heading:   "Unsubscribed",
			Message:   "You won't receive further emails from us.",
		}
		if err := tmpl.ExecuteTemplate(w, "layout", data); err != nil {
			log.Println("render unsubscribe:", err)
			http.Error(w, "internal error", http.StatusInternalServerError)
		}
	}
}
