// send-release emails a one-off announcement to confirmed subscribers.
//
// It is a dry run unless --send or --test-to is given. A send log table
// (release_sends) claims each (campaign, subscriber) pair before the email
// goes out, so re-running a campaign never emails anyone twice; a failed send
// releases its claim so the next run retries it.
package main

import (
	"context"
	"database/sql"
	_ "embed"
	"flag"
	"fmt"
	"html"
	"log"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/resend/resend-go/v3"

	"softstore/internal/db"
)

//go:embed release.html
var releaseHTML string

//go:embed release.txt
var releaseText string

const (
	fromAddr  = "Laudenslager Software <hello@lauden.dev>"
	replyAddr = "hello@lauden.dev"
)

type recipient struct {
	ID    int64
	Email string
	Token string
}

func ensureSendLog(conn *sql.DB) error {
	_, err := conn.Exec(`
	CREATE TABLE IF NOT EXISTS release_sends (
		campaign TEXT NOT NULL,
		subscriber_id INTEGER NOT NULL REFERENCES subscribers(id),
		claimed_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
		resend_id TEXT NOT NULL DEFAULT '',
		PRIMARY KEY (campaign, subscriber_id)
	)`)
	return err
}

func pendingRecipients(conn *sql.DB, campaign string) ([]recipient, error) {
	rows, err := conn.Query(`
		SELECT s.id, s.email, s.unsubscribe_token FROM subscribers s
		WHERE s.status = 'confirmed'
		AND NOT EXISTS (SELECT 1 FROM release_sends r WHERE r.campaign = ? AND r.subscriber_id = s.id)
		ORDER BY s.id`, campaign)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []recipient
	for rows.Next() {
		var r recipient
		if err := rows.Scan(&r.ID, &r.Email, &r.Token); err != nil {
			return nil, err
		}
		out = append(out, r)
	}
	return out, rows.Err()
}

func claim(conn *sql.DB, campaign string, id int64) (bool, error) {
	res, err := conn.Exec(`INSERT OR IGNORE INTO release_sends (campaign, subscriber_id) VALUES (?, ?)`, campaign, id)
	if err != nil {
		return false, err
	}
	n, err := res.RowsAffected()
	return n == 1, err
}

func release(conn *sql.DB, campaign string, id int64) error {
	_, err := conn.Exec(`DELETE FROM release_sends WHERE campaign = ? AND subscriber_id = ?`, campaign, id)
	return err
}

func recordResendID(conn *sql.DB, campaign string, id int64, resendID string) error {
	_, err := conn.Exec(`UPDATE release_sends SET resend_id = ? WHERE campaign = ? AND subscriber_id = ?`, resendID, campaign, id)
	return err
}

func render(tmpl, unsubURL, address string, escape bool) string {
	if escape {
		unsubURL = html.EscapeString(unsubURL)
		address = html.EscapeString(address)
	}
	return strings.NewReplacer("{{UNSUBSCRIBE_URL}}", unsubURL, "{{POSTAL_ADDRESS}}", address).Replace(tmpl)
}

func buildMessage(baseURL, to, token, subject, address string) *resend.SendEmailRequest {
	unsub := strings.TrimRight(baseURL, "/") + "/subscribe/unsubscribe?token=" + url.QueryEscape(token)
	return &resend.SendEmailRequest{
		From:    fromAddr,
		To:      []string{to},
		ReplyTo: replyAddr,
		Subject: subject,
		Html:    render(releaseHTML, unsub, address, true),
		Text:    render(releaseText, unsub, address, false),
		Headers: map[string]string{"List-Unsubscribe": "<" + unsub + ">"},
	}
}

func sendOne(client *resend.Client, req *resend.SendEmailRequest) (string, error) {
	sent, err := client.Emails.SendWithContext(context.Background(), req)
	if err != nil {
		return "", fmt.Errorf("resend send failed: %w", err)
	}
	return sent.Id, nil
}

func main() {
	campaign := flag.String("campaign", "", "campaign id, e.g. popfocus-launch-2026-10 (required)")
	address := flag.String("address", "", "postal address for the footer (required)")
	subject := flag.String("subject", "Pop Focus is now available", "email subject")
	testTo := flag.String("test-to", "", "send one test email to this address and exit (no send log)")
	send := flag.Bool("send", false, "send to all pending confirmed subscribers")
	flag.Parse()

	if *campaign == "" || *address == "" {
		log.Fatal("--campaign and --address are required")
	}
	if *send && *testTo != "" {
		log.Fatal("use either --send or --test-to, not both")
	}
	baseURL := strings.TrimRight(os.Getenv("BASE_URL"), "/")
	if baseURL == "" {
		log.Fatal("missing required env var: BASE_URL")
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
	if _, err := conn.Exec(`PRAGMA busy_timeout = 5000`); err != nil {
		log.Fatal(err)
	}
	if err := ensureSendLog(conn); err != nil {
		log.Fatal("send log: ", err)
	}
	pending, err := pendingRecipients(conn, *campaign)
	if err != nil {
		log.Fatal("recipients: ", err)
	}
	log.Printf("campaign %q: %d pending confirmed subscribers", *campaign, len(pending))

	var client *resend.Client
	if *send || *testTo != "" {
		key := os.Getenv("RESEND_API_KEY")
		if key == "" {
			log.Fatal("missing required env var: RESEND_API_KEY")
		}
		client = resend.NewClient(key)
	}

	switch {
	case *testTo != "":
		id, err := sendOne(client, buildMessage(baseURL, *testTo, "TEST", *subject, *address))
		if err != nil {
			log.Fatal(err)
		}
		log.Println("test email accepted, resend id:", id)
	case *send:
		sent, failed := 0, 0
		for _, r := range pending {
			ok, err := claim(conn, *campaign, r.ID)
			if err != nil {
				log.Fatal("claim: ", err)
			}
			if !ok {
				continue
			}
			id, err := sendOne(client, buildMessage(baseURL, r.Email, r.Token, *subject, *address))
			if err != nil {
				if rerr := release(conn, *campaign, r.ID); rerr != nil {
					log.Printf("could not release claim for subscriber id %d: %v", r.ID, rerr)
				}
				log.Printf("send failed for subscriber id %d: %v", r.ID, err)
				failed++
				continue
			}
			if err := recordResendID(conn, *campaign, r.ID, id); err != nil {
				log.Printf("could not record resend id for subscriber id %d: %v", r.ID, err)
			}
			log.Printf("sent to subscriber id %d (resend %s)", r.ID, id)
			sent++
			time.Sleep(600 * time.Millisecond)
		}
		log.Printf("done: %d sent, %d failed", sent, failed)
	default:
		log.Println("dry run (no emails sent). Subject:", *subject)
		fmt.Println("----- text body preview -----")
		fmt.Print(render(releaseText, baseURL+"/subscribe/unsubscribe?token=EXAMPLE", *address, false))
	}
}
