package main

import (
	"html/template"
	"log"
	"net/http"

	"github.com/stripe/stripe-go/v82"

	"softstore/internal/auth"
	"softstore/internal/cartsession"
	"softstore/internal/config"
	"softstore/internal/db"
	"softstore/internal/handlers"
	"softstore/internal/payments/stripeprovider"
	"softstore/web"
)

func main() {
	stripe.Key = config.StripeSecretKey()
	sessionSecret := config.SessionSecret()
	adminUsername := config.AdminUsername()
	passwordHash := config.AdminPasswordHash()
	auth.SecureCookies = config.SecureCookies()
	cartsession.SecureCookies = config.SecureCookies()
	baseURL := config.BaseURL()

        provider := stripeprovider.New()

	database, err := db.Open("softstore.db")
	if err != nil {
		log.Fatal(err)
	}
	defer database.Close()
	shopTmpl := template.Must(template.ParseFS(web.Templates,
		"templates/layout.html",
		"templates/shop.html",
	))
	adminTmpl := template.Must(template.ParseFS(web.Templates,
		"templates/layout.html",
		"templates/admin_new.html",
	))
	loginTmpl := template.Must(template.ParseFS(web.Templates,
		"templates/layout.html",
		"templates/admin_login.html",
	))
	thankYouTmpl := template.Must(template.ParseFS(web.Templates,
		"templates/layout.html",
		"templates/thank_you.html",
	))

	mux := http.NewServeMux()
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("ok"))
	})
	mux.HandleFunc("/", handlers.Shop(database, shopTmpl))
	mux.HandleFunc("POST /checkout/{slug}", handlers.Checkout(database, provider, baseURL))
	mux.HandleFunc("POST /cart/add/{slug}", handlers.AddToCart(database))
	mux.HandleFunc("GET /thank-you", func(w http.ResponseWriter, r *http.Request) {
		thankYouTmpl.ExecuteTemplate(w, "layout", handlers.ShopData{Title: "Thank You"})
	})

	mux.HandleFunc("GET /admin/login", handlers.AdminLoginForm(loginTmpl))
	mux.HandleFunc("POST /admin/login", handlers.AdminLoginSubmit(loginTmpl, adminUsername, passwordHash, sessionSecret))
	mux.HandleFunc("POST /admin/logout", handlers.AdminLogout)

	mux.HandleFunc("GET /admin/products/new", handlers.RequireAdmin(sessionSecret, handlers.AdminNew(adminTmpl)))
	mux.HandleFunc("POST /admin/products", handlers.RequireAdmin(sessionSecret, handlers.AdminCreateProduct(database, provider)))

	mux.Handle("/static/", http.FileServer(http.FS(web.Static)))

	log.Println("listening on :38217 (http, behind caddy)")
	if err := http.ListenAndServe(":38217", mux); err != nil {
		log.Fatal(err)
	}
}
