// weekly-subscriber-report emails the current confirmed-subscriber
// count to the site owner, on a schedule via systemd timer.
//
// Run: RESEND_API_KEY=re_... go run ./cmd/weekly-subscriber-report
package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/resend/resend-go/v3"

	"softstore/internal/db"
)

func main() {
	resendKey := os.Getenv("RESEND_API_KEY")
	if resendKey == "" {
		log.Fatal("missing required env var: RESEND_API_KEY")
	}

	reportTo := os.Getenv("REPORT_TO_EMAIL")
	if reportTo == "" {
		log.Fatal("missing required env var: REPORT_TO_EMAIL")
	}

	dbPath := os.Getenv("SOFTSTORE_DB_PATH")
	if dbPath == "" {
		dbPath = "softstore.db"
	}

	conn, err := db.Open(dbPath)
	if err != nil {
		log.Fatal("open db: ", err)
	}
	defer conn.Close()

	count, err := db.CountConfirmedSubscribers(conn)
	if err != nil {
		log.Fatal("count confirmed subscribers: ", err)
	}

	weekOf := time.Now().Format("January 2, 2006")
	subject := fmt.Sprintf("Subscriber count: %d (week of %s)", count, weekOf)
	text := fmt.Sprintf("You currently have %d confirmed subscribers.\n", count)
	html := fmt.Sprintf(`<p>You currently have <strong>%d</strong> confirmed subscribers.</p>`, count)

	client := resend.NewClient(resendKey)
	params := &resend.SendEmailRequest{
		From:    "hello@lauden.dev",
		To:      []string{reportTo},
		Subject: subject,
		Html:    html,
		Text:    text,
	}

	sent, err := client.Emails.SendWithContext(context.Background(), params)
	if err != nil {
		log.Fatal("resend send failed: ", err)
	}
	log.Println("resend accepted, id:", sent.Id)
}
