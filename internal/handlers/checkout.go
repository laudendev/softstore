package handlers

import (
	"database/sql"
	"errors"
	"log"
	"net/http"

	"github.com/stripe/stripe-go/v82"
	"github.com/stripe/stripe-go/v82/checkout/session"

	"softstore/internal/db"
)

func Checkout(conn *sql.DB, baseURL string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		slug := r.PathValue("slug")

		product, err := db.GetProductBySlug(conn, slug)
		if err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				http.NotFound(w, r)
				return
			}
			log.Println("get product by slug:", err)
			http.Error(w, "internal error", http.StatusInternalServerError)
			return
		}

		params := &stripe.CheckoutSessionParams{
			Mode: stripe.String(string(stripe.CheckoutSessionModePayment)),
			LineItems: []*stripe.CheckoutSessionLineItemParams{
				{
					Price:    stripe.String(product.StripePriceID),
					Quantity: stripe.Int64(1),
				},
			},
			Metadata: map[string]string{
				"product": product.ProductCode,
				"seats":   "1",
			},
			SuccessURL: stripe.String(baseURL + "/thank-you"),
			CancelURL:  stripe.String(baseURL + "/"),
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
