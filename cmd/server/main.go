package main

import (
	"html/template"
	"log"
	"net/http"

	"github.com/stripe/stripe-go/v82"

	"softstore/internal/config"
	"softstore/internal/db"
	"softstore/internal/handlers"
)

func main() {
	stripe.Key = config.StripeSecretKey()
	sessionSecret := config.SessionSecret()
	adminUsername := config.AdminUsername()
	passwordHash := config.AdminPasswordHash()

	database, err := db.Open("softstore.db")
	if err != nil {
		log.Fatal(err)
	}
	defer database.Close()

	homeTmpl := template.Must(template.ParseFiles(
		"web/templates/layout.html",
		"web/templates/home.html",
	))
	adminTmpl := template.Must(template.ParseFiles(
		"web/templates/layout.html",
		"web/templates/admin_new.html",
	))
	loginTmpl := template.Must(template.ParseFiles(
		"web/templates/layout.html",
		"web/templates/admin_login.html",
	))
	thankYouTmpl := template.Must(template.ParseFiles(
		"web/templates/layout.html",
		"web/templates/thank_you.html",
	))

	mux := http.NewServeMux()
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("ok"))
	})
	mux.HandleFunc("/", handlers.Home(database, homeTmpl))
	mux.HandleFunc("POST /checkout/{slug}", handlers.Checkout(database))
	mux.HandleFunc("GET /thank-you", func(w http.ResponseWriter, r *http.Request) {
		thankYouTmpl.ExecuteTemplate(w, "layout", handlers.HomeData{Title: "Thank You"})
	})

	mux.HandleFunc("GET /admin/login", handlers.AdminLoginForm(loginTmpl))
	mux.HandleFunc("POST /admin/login", handlers.AdminLoginSubmit(loginTmpl, adminUsername, passwordHash, sessionSecret))
	mux.HandleFunc("POST /admin/logout", handlers.AdminLogout)

	mux.HandleFunc("GET /admin/products/new", handlers.RequireAdmin(sessionSecret, handlers.AdminNew(adminTmpl)))
	mux.HandleFunc("POST /admin/products", handlers.RequireAdmin(sessionSecret, handlers.AdminCreateProduct(database)))

	mux.Handle("/static/", http.StripPrefix("/static/", http.FileServer(http.Dir("web/static"))))

	log.Println("listening on :8443 (https)")
	if err := http.ListenAndServeTLS(":8443", "localhost-cert.pem", "localhost-key.pem", mux); err != nil {
		log.Fatal(err)
	}
}
