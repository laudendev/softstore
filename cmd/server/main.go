package main

import (
	"html/template"
	"log"
	"net/http"

	"softstore/internal/db"
	"softstore/internal/handlers"
)

func main() {
	database, err := db.Open("softstore.db")
	if err != nil {
		log.Fatal(err)
	}
	defer database.Close()

	tmpl := template.Must(template.ParseFiles(
		"web/templates/layout.html",
		"web/templates/home.html",
	))

	mux := http.NewServeMux()
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("ok"))
	})
	mux.HandleFunc("/", handlers.Home(database, tmpl))
	mux.Handle("/static/", http.StripPrefix("/static/", http.FileServer(http.Dir("web/static"))))

	log.Println("listening on :8080")
	if err := http.ListenAndServe(":8080", mux); err != nil {
		log.Fatal(err)
	}
}
