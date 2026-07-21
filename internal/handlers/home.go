package handlers

import (
	"database/sql"
	"html/template"
	"log"
	"net/http"

	"softstore/internal/db"
)

type HomeData struct {
	Title    string
	Products interface{}
}

func Home(conn *sql.DB, tmpl *template.Template) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		products, err := db.ListProducts(conn)
		if err != nil {
			log.Println("list products:", err)
			http.Error(w, "internal error", http.StatusInternalServerError)
			return
		}

		data := HomeData{
			Title:    "Home",
			Products: products,
		}

		if err := tmpl.ExecuteTemplate(w, "layout", data); err != nil {
			log.Println("render home:", err)
			http.Error(w, "internal error", http.StatusInternalServerError)
		}
	}
}
