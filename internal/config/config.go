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

func AdminPasswordHash() string {
	return requireEnv("ADMIN_PASSWORD_HASH")
}

func AdminUsername() string {
        return requireEnv("ADMIN_USERNAME")
}

func SessionSecret() []byte {
	return []byte(requireEnv("SESSION_SECRET"))
}

func SecureCookies() bool {
	return os.Getenv("SECURE_COOKIES") == "true"
}

func BaseURL() string {
	return requireEnv("BASE_URL")
}

// InternalAPISecret authenticates service-to-service requests (e.g. from
// Quartermaster) to softstore's /internal/* endpoints. Not a user-facing
// credential — a static shared secret checked via constant-time compare.
func InternalAPISecret() string {
	return requireEnv("INTERNAL_API_SECRET")
}


// QuartermasterInternalURL is Quartermaster's WireGuard-only internal API
// base URL, used by softstore to poll checkout session fulfillment status.
func QuartermasterInternalURL() string {
	return requireEnv("QUARTERMASTER_INTERNAL_URL")
}
