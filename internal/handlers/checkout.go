package handlers

import (
	"database/sql"
	"log"
	"net/http"

	"github.com/stripe/stripe-go/v82"
	"github.com/stripe/stripe-go/v82/checkout/session"

	"softstore/internal/db"
)

func Checkout(conn *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		slug := r.PathValue("slug")

		products, err := db.ListProducts(conn)
		if err != nil {
			log.Println("list products:", err)
			http.Error(w, "internal error", http.StatusInternalServerError)
			return
		}

		var priceID, productCode string
		found := false
		for _, p := range products {
			if p.Slug == slug {
				priceID = p.StripePriceID
				productCode = p.ProductCode
				found = true
				break
			}
		}
		if !found {
			http.NotFound(w, r)
			return
		}

		params := &stripe.CheckoutSessionParams{
			Mode: stripe.String(string(stripe.CheckoutSessionModePayment)),
			LineItems: []*stripe.CheckoutSessionLineItemParams{
				{
					Price:    stripe.String(priceID),
					Quantity: stripe.Int64(1),
				},
			},
			Metadata: map[string]string{
				"product": productCode,
				"seats":   "1",
			},
			SuccessURL: stripe.String("http://localhost:8080/thank-you"),
			CancelURL:  stripe.String("http://localhost:8080/"),
		}

		s, err := session.New(params)
		if err != nil {
			log.Println("stripe session create:", err)
			http.Error(w, "checkout error", http.StatusInternalServerError)
			return
		}

		http.Redirect(w, r, s.URL, http.StatusSeeOther)
	}
}
