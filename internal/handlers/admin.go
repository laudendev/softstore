package handlers

import (
	"database/sql"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"strconv"
	"math"

	"softstore/internal/db"
	"softstore/internal/models"
)

func AdminNew(tmpl *template.Template) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		data := HomeData{Title: "Add Product"}
		if err := tmpl.ExecuteTemplate(w, "layout", data); err != nil {
			log.Println("render admin_new:", err)
			http.Error(w, "internal error", http.StatusInternalServerError)
		}
	}
}

func AdminCreateProduct(conn *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if err := r.ParseForm(); err != nil {
			http.Error(w, "bad form", http.StatusBadRequest)
			return
		}

		priceDollars, err := strconv.ParseFloat(r.FormValue("price"), 64)
		if err != nil {
			fmt.Fprintf(w, `<p class="error">Invalid price.</p>`)
			return
		}

		code := r.FormValue("product_code")
		if len(code) != 4 {
			fmt.Fprintf(w, `<p class="error">Product code must be exactly 4 characters.</p>`)
			return
		}

		p := &models.Product{
			Name:          r.FormValue("name"),
			Slug:          r.FormValue("slug"),
			Description:   r.FormValue("description"),
			PriceCents:    int64(math.Round(priceDollars * 100)),
			StripePriceID: r.FormValue("stripe_price_id"),
			ProductCode:   code,
			StubURL:       r.FormValue("stub_url"),
		}

		if err := db.CreateProduct(conn, p); err != nil {
			log.Println("create product:", err)
			fmt.Fprintf(w, `<p class="error">Failed to create product: %s</p>`, err)
			return
		}

		fmt.Fprintf(w, `<p class="success">Created "%s" successfully.</p>`, p.Name)
	}
}
