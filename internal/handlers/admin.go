package handlers

import (
	"database/sql"
	"fmt"
	"html/template"
	"log"
	"math"
	"net/http"
	"strconv"
	"crypto/subtle"


	"softstore/internal/db"
	"softstore/internal/models"
	"softstore/internal/auth"
	"softstore/internal/payments"
)

func AdminNew(tmpl *template.Template) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		data := ShopData{Title: "add product"}
		if err := tmpl.ExecuteTemplate(w, "layout", data); err != nil {
			log.Println("render admin_new:", err)
			http.Error(w, "internal error", http.StatusInternalServerError)
		}
	}
}

func AdminLoginForm(tmpl *template.Template) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		data := struct {
			Title     string
			Error     string
			CartCount int64
			ShowCart bool
		}{Title: "Admin Login"}
		if err := tmpl.ExecuteTemplate(w, "layout", data); err != nil {
			log.Println("render admin_login:", err)
			http.Error(w, "internal error", http.StatusInternalServerError)
		}
	}
}

func AdminLoginSubmit(tmpl *template.Template, username, passwordHash string, sessionSecret []byte) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if err := r.ParseForm(); err != nil {
			http.Error(w, "bad form", http.StatusBadRequest)
			return
		}
		submittedUsername := r.FormValue("username")
		password := r.FormValue("password")

		validUsername := subtle.ConstantTimeCompare([]byte(submittedUsername), []byte(username)) == 1
		validPassword := auth.CheckPassword(passwordHash, password)

		if !validUsername || !validPassword {
			data := struct {
				Title     string
				Error     string
				CartCount int64
				ShowCart bool
			}{Title: "Admin Login", Error: "Incorrect username or password."}
			tmpl.ExecuteTemplate(w, "layout", data)
			return
		}
		
		auth.SetSessionCookie(w, sessionSecret)
		http.Redirect(w, r, "/admin/products/new", http.StatusSeeOther)
	}
}


func AdminLogout(w http.ResponseWriter, r *http.Request) {
	auth.ClearSessionCookie(w)
	http.Redirect(w, r, "/admin/login", http.StatusSeeOther)
}

func AdminCreateProduct(conn *sql.DB, provider payments.Provider) http.HandlerFunc {
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
		priceCents := int64(math.Round(priceDollars * 100))

		code := r.FormValue("product_code")
		if len(code) != 4 {
			fmt.Fprintf(w, `<p class="error">Product code must be exactly 4 characters.</p>`)
			return
		}

		seats, err := strconv.ParseInt(r.FormValue("seats"), 10, 64)
		if err != nil || seats < 1 {
			fmt.Fprintf(w, `<p class="error">Seats must be a whole number of 1 or more.</p>`)
			return
		}

		name := r.FormValue("name")
		description := r.FormValue("description")
		taxCode := r.FormValue("tax_code")

		registered, err := provider.RegisterItem(payments.SellableItem{
			Name:        name,
			Description: description,
			PriceCents:  priceCents,
			Currency:    "usd",
			TaxCategory: taxCode,
		})
		if err != nil {
			log.Println("provider register item:", err)
			fmt.Fprintf(w, `<p class="error">Failed to register product with payment provider: %s</p>`, err)
			return
		}

		p := &models.Product{
			Name:          name,
			Slug:          r.FormValue("slug"),
			Description:   description,
			PriceCents:    priceCents,
			StripePriceID: registered.ProviderItemID,
			StripeProductID: registered.ProviderProductID,
			ProductCode:   code,
			StubURL:       r.FormValue("stub_url"),
			TaxCode:       taxCode,
			Seats:         seats,
		}

		if err := db.CreateProduct(conn, p); err != nil {
			log.Println("create product:", err)
			fmt.Fprintf(w, `<p class="error">Failed to save product locally: %s</p>`, err)
			return
		}

		fmt.Fprintf(w, `<p class="success">Created "%s" successfully. (Provider item: %s)</p>`, p.Name, registered.ProviderItemID)
	}
}
