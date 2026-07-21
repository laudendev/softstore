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

	mux := http.NewServeMux()
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("ok"))
	})
	mux.HandleFunc("/", handlers.Home(database, homeTmpl))
	mux.HandleFunc("GET /admin/products/new", handlers.AdminNew(adminTmpl))
	mux.HandleFunc("POST /admin/products", handlers.AdminCreateProduct(database))
	mux.Handle("/static/", http.StripPrefix("/static/", http.FileServer(http.Dir("web/static"))))

	log.Println("listening on :8080")
	if err := http.ListenAndServe(":8080", mux); err != nil {
		log.Fatal(err)
	}
}
