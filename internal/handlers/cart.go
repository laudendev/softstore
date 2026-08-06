package handlers

import (
	"database/sql"
	"errors"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"strconv"

	"softstore/internal/cartsession"
	"softstore/internal/db"
	"softstore/internal/payments"
)

// parseSeatsForm reads the "seats" form value from an add-to-cart
// request, defaulting to 1 (and clamping to the same [1, 24] range
// enforced elsewhere) if missing, empty, or unparseable — a bare "+"
// click with no seat selection should behave exactly like a 1-seat
// purchase, not fail the request.
func parseSeatsForm(r *http.Request) int64 {
	seats, err := strconv.ParseInt(r.FormValue("seats"), 10, 64)
	if err != nil || seats < 1 {
		return 1
	}
	if seats > 24 {
		return 24
	}
	return seats
}

// AddToCart handles POST /cart/add/{slug}. It adds one unit of the product
// to the caller's cart and responds with an HTMX fragment showing the
// updated cart item count, for out-of-band swap into the nav.
func AddToCart(conn *sql.DB, tmpl *template.Template, provider payments.Provider) http.HandlerFunc {
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

		seats := parseSeatsForm(r)
		if _, err := GetOrCreatePriceForSeats(conn, provider, product, seats); err != nil {
			log.Println("add to cart, resolve price for seats:", err)
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

		if err := db.AddCartItem(conn, cart.ID, product.ID, seats, 1); err != nil {
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
		fmt.Fprintf(w, `<span id="cart-count" class="cart-count-badge" hx-swap-oob="true">%d</span>`, updated.ItemCount())
		if err := tmpl.ExecuteTemplate(w, "cart-drawer-content", updated); err != nil {
			log.Println("render cart drawer:", err)
			http.Error(w, "internal error", http.StatusInternalServerError)
			return
		}
	}
}

// GetCart handles GET /cart. It renders the cart drawer's contents for
// the caller's cart, as an HTMX fragment.
func GetCart(conn *sql.DB, tmpl *template.Template) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		token := cartsession.Token(w, r)

		cart, err := db.GetCartWithItems(conn, token)
		if err != nil {
			log.Println("get cart:", err)
			http.Error(w, "internal error", http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		if err := tmpl.ExecuteTemplate(w, "cart-drawer-content", cart); err != nil {
			log.Println("render cart drawer:", err)
			http.Error(w, "internal error", http.StatusInternalServerError)
			return
		}
	}
}

// RemoveFromCart handles POST /cart/remove/{slug}. It removes the given
// product from the caller's cart and re-renders the drawer fragment.
func RemoveFromCart(conn *sql.DB, tmpl *template.Template) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		slug := r.PathValue("slug")
		seats := parseSeatsForm(r)

		product, err := db.GetProductBySlug(conn, slug)
		if err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				http.NotFound(w, r)
				return
			}
			log.Println("remove from cart, get product:", err)
			http.Error(w, "internal error", http.StatusInternalServerError)
			return
		}

		token := cartsession.Token(w, r)
		cart, err := db.GetOrCreateCart(conn, token)
		if err != nil {
			log.Println("remove from cart, get cart:", err)
			http.Error(w, "internal error", http.StatusInternalServerError)
			return
		}

		if err := db.RemoveCartItem(conn, cart.ID, product.ID, seats); err != nil {
			log.Println("remove from cart, delete item:", err)
			http.Error(w, "internal error", http.StatusInternalServerError)
			return
		}

		updated, err := db.GetCartWithItems(conn, token)
		if err != nil {
			log.Println("remove from cart, reload cart:", err)
			http.Error(w, "internal error", http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		fmt.Fprintf(w, `<span id="cart-count" class="cart-count-badge" hx-swap-oob="true">%d</span>`, updated.ItemCount())
		if err := tmpl.ExecuteTemplate(w, "cart-drawer-content", updated); err != nil {
			log.Println("render cart drawer:", err)
			http.Error(w, "internal error", http.StatusInternalServerError)
			return
		}
	}
}
