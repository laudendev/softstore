package handlers

import (
	"database/sql"
	"encoding/json"
	"errors"
	"log"
	"net/http"

	"softstore/internal/db"
)

// productByPriceResponse is the JSON shape returned by
// GET /internal/products/by-price/{price_id}.
type productByPriceResponse struct {
	ProductCode string `json:"product_code"`
	Name        string `json:"name"`
	Seats       int64  `json:"seats"`
}

// GetProductByPrice handles GET /internal/products/by-price/{price_id}.
// It's a service-to-service endpoint (guarded by RequireInternalSecret)
// that lets Quartermaster resolve a Stripe Price ID from a checkout
// session's line items back to softstore's product code, so it knows
// which license to issue.
func GetProductByPrice(conn *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		priceID := r.PathValue("price_id")

		product, err := db.GetProductByStripePriceID(conn, priceID)
		if err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				http.NotFound(w, r)
				return
			}
			log.Println("internal get product by price:", err)
			http.Error(w, "internal error", http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(productByPriceResponse{
			ProductCode: product.ProductCode,
			Name:        product.Name,
			Seats:       product.Seats,
		})
	}
}

// clearCartRequest is the JSON body for POST /internal/cart/clear.
type clearCartRequest struct {
	CartToken string `json:"cart_token"`
}

// ClearCart handles POST /internal/cart/clear. It's a service-to-service
// endpoint (guarded by RequireInternalSecret) called by Quartermaster
// after a purchase has been successfully fulfilled, so a completed
// cart doesn't linger and reappear on the buyer's next visit.
func ClearCart(conn *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var body clearCartRequest
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil || body.CartToken == "" {
			http.Error(w, "bad request", http.StatusBadRequest)
			return
		}

		if err := db.ClearCart(conn, body.CartToken); err != nil {
			log.Println("internal clear cart:", err)
			http.Error(w, "internal error", http.StatusInternalServerError)
			return
		}

		w.WriteHeader(http.StatusOK)
	}
}
