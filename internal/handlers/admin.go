package handlers

import (
	"database/sql"
	"fmt"
	"html/template"
	"log"
	"math"
	"net/http"
	"strconv"

	"github.com/stripe/stripe-go/v82"
	stripeprice "github.com/stripe/stripe-go/v82/price"
	stripeproduct "github.com/stripe/stripe-go/v82/product"

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
		priceCents := int64(math.Round(priceDollars * 100))

		code := r.FormValue("product_code")
		if len(code) != 4 {
			fmt.Fprintf(w, `<p class="error">Product code must be exactly 4 characters.</p>`)
			return
		}

		name := r.FormValue("name")
		description := r.FormValue("description")
		taxCode := r.FormValue("tax_code")

		// Create the Stripe Product.
		stripeProd, err := stripeproduct.New(&stripe.ProductParams{
			Name:        stripe.String(name),
			Description: stripe.String(description),
			TaxCode:     stripe.String(taxCode),
		})
		if err != nil {
			log.Println("stripe product create:", err)
			fmt.Fprintf(w, `<p class="error">Failed to create Stripe product: %s</p>`, err)
			return
		}

		// Create the Stripe Price, attached to that Product.
		stripePrice, err := stripeprice.New(&stripe.PriceParams{
			Product:    stripe.String(stripeProd.ID),
			UnitAmount: stripe.Int64(priceCents),
			Currency:   stripe.String(string(stripe.CurrencyUSD)),
		})
		if err != nil {
			log.Println("stripe price create:", err)
			fmt.Fprintf(w, `<p class="error">Failed to create Stripe price: %s</p>`, err)
			return
		}

		p := &models.Product{
			Name:          name,
			Slug:          r.FormValue("slug"),
			Description:   description,
			PriceCents:    priceCents,
			StripePriceID: stripePrice.ID,
			ProductCode:   code,
			StubURL:       r.FormValue("stub_url"),
			TaxCode:       taxCode,
		}

		if err := db.CreateProduct(conn, p); err != nil {
			log.Println("create product:", err)
			fmt.Fprintf(w, `<p class="error">Failed to save product locally: %s</p>`, err)
			return
		}

		fmt.Fprintf(w, `<p class="success">Created "%s" successfully. (Stripe Price: %s)</p>`, p.Name, stripePrice.ID)
	}
}
