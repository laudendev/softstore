package config

import (
   "log"
   "os"
)

func requireEnv(key string) string {
     v := os.Getenv(key)
     if v == "" {
	log.Fatalf("missing required env var: %s", key)
     }
     return v
}

func StripeSecretKey() string {
     return requireEnv("STRIPE_SECRET_KEY")
}
