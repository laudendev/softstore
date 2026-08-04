package handlers

import (
	"database/sql"
	"errors"
	"fmt"
	"html/template"
	"log"
	"net/http"

	"softstore/internal/cartsession"
	"softstore/internal/db"
)

// AddToCart handles POST /cart/add/{slug}. It adds one unit of the product
// to the caller's cart and responds with an HTMX fragment showing the
// updated cart item count, for out-of-band swap into the nav.
func AddToCart(conn *sql.DB, tmpl *template.Template) http.HandlerFunc {
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
		fmt.Fprintf(w, `<span id="cart-count" hx-swap-oob="true">%d</span>`, updated.ItemCount())
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

		if err := db.RemoveCartItem(conn, cart.ID, product.ID); err != nil {
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
		fmt.Fprintf(w, `<span id="cart-count" hx-swap-oob="true">%d</span>`, updated.ItemCount())
		if err := tmpl.ExecuteTemplate(w, "cart-drawer-content", updated); err != nil {
			log.Println("render cart drawer:", err)
			http.Error(w, "internal error", http.StatusInternalServerError)
			return
		}
	}
}
