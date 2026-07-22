package handlers

import (
	"database/sql"
	"html/template"
	"log"
	"net/http"

	"softstore/internal/db"
)

type ShopData struct {
	Title    string
	Products interface{}
}

func Shop(conn *sql.DB, tmpl *template.Template) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		products, err := db.ListProducts(conn)
		if err != nil {
			log.Println("list products:", err)
			http.Error(w, "internal error", http.StatusInternalServerError)
			return
		}

		data := ShopData{
			Title:    "Shop",
			Products: products,
		}

		if err := tmpl.ExecuteTemplate(w, "layout", data); err != nil {
			log.Println("render shop:", err)
			http.Error(w, "internal error", http.StatusInternalServerError)
		}
	}
}
