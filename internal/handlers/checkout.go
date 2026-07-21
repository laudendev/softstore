package handlers

import (
	"database/sql"
	"errors"
	"log"
	"net/http"

	"softstore/internal/db"
	"softstore/internal/payments"
)

func Checkout(conn *sql.DB, provider payments.Provider, baseURL string) http.HandlerFunc {
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

		purchase, err := provider.StartPurchase(payments.PurchaseRequest{
			ProviderItemID: product.StripePriceID,
			Quantity:       1,
			Metadata: map[string]string{
				"product": product.ProductCode,
				"seats":   "1",
			},
			SuccessURL: baseURL + "/thank-you",
			CancelURL:  baseURL + "/",
		})
		if err != nil {
			log.Println("start purchase:", err)
			http.Error(w, "checkout error", http.StatusInternalServerError)
			return
		}

		http.Redirect(w, r, purchase.RedirectURL, http.StatusSeeOther)
	}
}
