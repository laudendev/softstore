// backfill-tax-behavior is a one-time operational script that sets
// tax_behavior=exclusive on every existing product's Stripe Price,
// since prices created before Stripe Tax was enabled default to
// "unspecified" and won't calculate tax correctly otherwise.
//
// Run once: STRIPE_SECRET_KEY=sk_... go run ./cmd/backfill-tax-behavior
package main

import (
	"log"
	"os"

	"github.com/stripe/stripe-go/v82"
	"github.com/stripe/stripe-go/v82/price"

	"softstore/internal/db"
)

func main() {
	secretKey := os.Getenv("STRIPE_SECRET_KEY")
	if secretKey == "" {
		log.Fatal("missing required env var: STRIPE_SECRET_KEY")
	}
	stripe.Key = secretKey

	dbPath := os.Getenv("SOFTSTORE_DB_PATH")
	if dbPath == "" {
		dbPath = "softstore.db"
	}

	conn, err := db.Open(dbPath)
	if err != nil {
		log.Fatal("open db: ", err)
	}
	defer conn.Close()

	products, err := db.ListProducts(conn)
	if err != nil {
		log.Fatal("list products: ", err)
	}

	log.Printf("found %d product(s) to check\n", len(products))

	for _, p := range products {
		existing, err := price.Get(p.StripePriceID, nil)
		if err != nil {
			log.Printf("skip %s (%s): fetch failed: %v\n", p.Name, p.StripePriceID, err)
			continue
		}

		if existing.TaxBehavior != stripe.PriceTaxBehaviorUnspecified {
			log.Printf("skip %s (%s): already set to %q\n", p.Name, p.StripePriceID, existing.TaxBehavior)
			continue
		}

		_, err = price.Update(p.StripePriceID, &stripe.PriceParams{
			TaxBehavior: stripe.String(string(stripe.PriceTaxBehaviorExclusive)),
		})
		if err != nil {
			log.Printf("FAILED %s (%s): %v\n", p.Name, p.StripePriceID, err)
			continue
		}
		log.Printf("updated %s (%s) -> exclusive\n", p.Name, p.StripePriceID)
	}

	log.Println("done")
}
