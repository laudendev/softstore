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
