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
	CartCount int64
}


// cartCountForRequest looks up the caller's current cart item count for
// display in the layout's header badge, without setting a cart cookie
// if the visitor doesn't have one yet (a bare page view shouldn't
// create a cart — only adding an item should).
func cartCountForRequest(conn *sql.DB, r *http.Request) int64 {
	cookie, err := r.Cookie("softstore_cart")
	if err != nil || cookie.Value == "" {
		return 0
	}
	cart, err := db.GetCartWithItems(conn, cookie.Value)
	if err != nil {
		return 0
	}
	return cart.ItemCount()
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
			Title:    "shop",
			Products: products,
			CartCount: cartCountForRequest(conn, r),
		}

		if err := tmpl.ExecuteTemplate(w, "layout", data); err != nil {
			log.Println("render shop:", err)
			http.Error(w, "internal error", http.StatusInternalServerError)
		}
	}
}
