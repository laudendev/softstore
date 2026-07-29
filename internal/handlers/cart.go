package handlers

import (
	"database/sql"
	"errors"
	"fmt"
	"log"
	"net/http"

	"softstore/internal/cartsession"
	"softstore/internal/db"
)

// AddToCart handles POST /cart/add/{slug}. It adds one unit of the product
// to the caller's cart and responds with an HTMX fragment showing the
// updated cart item count, for out-of-band swap into the nav.
func AddToCart(conn *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		slug := r.PathValue("slug")

		product, err := db.GetProductBySlug(conn, slug)
		if err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				http.NotFound(w, r)
				return
			}
			log.Println("add to cart, get product:", err)
			http.Error(w, "internal error", http.StatusInternalServerError)
			return
		}

		token := cartsession.Token(w, r)
		cart, err := db.GetOrCreateCart(conn, token)
		if err != nil {
			log.Println("add to cart, get cart:", err)
			http.Error(w, "internal error", http.StatusInternalServerError)
			return
		}

		if err := db.AddCartItem(conn, cart.ID, product.ID, 1); err != nil {
			log.Println("add to cart, add item:", err)
			http.Error(w, "internal error", http.StatusInternalServerError)
			return
		}

		updated, err := db.GetCartWithItems(conn, token)
		if err != nil {
			log.Println("add to cart, reload cart:", err)
			http.Error(w, "internal error", http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		fmt.Fprintf(w, `<span id="cart-count">%d</span>`, updated.ItemCount())
	}
}
