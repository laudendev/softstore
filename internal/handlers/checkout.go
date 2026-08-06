package handlers

import (
	"database/sql"
	"errors"
	"log"
	"net/http"
	"strings"

	"softstore/internal/cartsession"
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
        seats := parseSeatsForm(r)
		priceID, err := GetOrCreatePriceForSeats(conn, provider, product, seats)
		if err != nil {
			log.Println("checkout, resolve price for seats:", err)
			http.Error(w, "internal error", http.StatusInternalServerError)
			return
		}

		purchase, err := provider.StartPurchase(payments.PurchaseRequest{
			LineItems: []payments.LineItem{
				{ProviderItemID: priceID, Quantity: 1},
			},
			Metadata: map[string]string{
				"product": product.ProductCode,
			},
			SuccessURL: baseURL + "/thank-you?session_id={CHECKOUT_SESSION_ID}",
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

// CartCheckout handles POST /checkout. It builds a single Stripe session
// from every item in the caller's cart and redirects to it.
func CartCheckout(conn *sql.DB, provider payments.Provider, baseURL string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		token := cartsession.Token(w, r)

		cart, err := db.GetCartWithItems(conn, token)
		if err != nil {
			log.Println("cart checkout, get cart:", err)
			http.Error(w, "internal error", http.StatusInternalServerError)
			return
		}

		if len(cart.Items) == 0 {
			http.Error(w, "cart is empty", http.StatusBadRequest)
			return
		}

		lineItems := make([]payments.LineItem, 0, len(cart.Items))
		productCodes := make([]string, 0, len(cart.Items))
		for _, item := range cart.Items {
			priceID, err := GetOrCreatePriceForSeats(conn, provider, &item.Product, item.Seats)
			if err != nil {
				log.Println("cart checkout, resolve price for seats:", err)
				http.Error(w, "internal error", http.StatusInternalServerError)
				return
			}
			lineItems = append(lineItems, payments.LineItem{
				ProviderItemID: priceID,
				Quantity:       item.Quantity,
			})
			productCodes = append(productCodes, item.Product.ProductCode)
		}
		
		purchase, err := provider.StartPurchase(payments.PurchaseRequest{
			LineItems: lineItems,
			Metadata: map[string]string{
				"cart_token":    token,
				"product_codes": strings.Join(productCodes, ","),
			},
			SuccessURL: baseURL + "/thank-you?session_id={CHECKOUT_SESSION_ID}",
			CancelURL:  baseURL + "/",
		})
		if err != nil {
			log.Println("cart checkout, start purchase:", err)
			http.Error(w, "checkout error", http.StatusInternalServerError)
			return
		}

		http.Redirect(w, r, purchase.RedirectURL, http.StatusSeeOther)
	}
}
