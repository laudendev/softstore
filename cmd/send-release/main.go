// send-release emails a one-off notice to confirmed subscribers and, with
// --buyers, to everyone who paid for a product in Stripe live mode.
//
// It is a dry run unless --send or --test-to is given. A send log table
// (notice_sends) claims each (campaign, recipient) pair before the email goes
// out, so re-running a campaign never emails anyone twice; a failed send
// releases its claim so the next run retries it. Logs and the send log hold
// subscriber ids and short hashes, never addresses.
//
// Subscribers get an unsubscribe link and a List-Unsubscribe header. A buyer
// who is not a confirmed subscriber gets a one-time service notice about the
// product they bought instead: they are on no list, so there is nothing to
// unsubscribe from. A buyer who is also a subscriber gets one email, the
// subscriber version.
package main

import (
	"context"
	"crypto/sha256"
	"database/sql"
	_ "embed"
	"encoding/hex"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"html"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"strconv"
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
	fromAddr       = "Laudenslager Software <hello@lauden.dev>"
	replyAddr      = "hello@lauden.dev"
	defaultSubject = "Pop Focus 0.1.21: please install this update once by hand"
	stripeAPI      = "https://api.stripe.com"
)

// Footers. {{UNSUBSCRIBE_URL}} and {{POSTAL_ADDRESS}} are filled in afterwards.
const (
	subscriberFooterText = "You are receiving this because you subscribed to release updates at\n" +
		"lauden.dev. Unsubscribe: {{UNSUBSCRIBE_URL}}\n" +
		"Laudenslager Software, LLC, {{POSTAL_ADDRESS}}"
	buyerFooterText = "You are receiving this one-time notice because you bought a Pop Focus\n" +
		"license. You are not on our mailing list.\n" +
		"Laudenslager Software, LLC, {{POSTAL_ADDRESS}}"
	subscriberFooterHTML = `You are receiving this because you subscribed to release updates at lauden.dev. ` +
		`<a href="{{UNSUBSCRIBE_URL}}" style="color:#6B7280;">Unsubscribe</a><br>` +
		`Laudenslager Software, LLC, {{POSTAL_ADDRESS}}`
	buyerFooterHTML = `You are receiving this one-time notice because you bought a Pop Focus license. ` +
		`You are not on our mailing list.<br>Laudenslager Software, LLC, {{POSTAL_ADDRESS}}`
)

type recipient struct {
	// Key identifies the recipient in the send log and in logs: never an address.
	Key   string
	Email string
	// Token is the unsubscribe token; empty for a buyer who is not a subscriber.
	Token string
}

func (r recipient) isSubscriber() bool { return r.Token != "" }

func ensureSendLog(conn *sql.DB) error {
	_, err := conn.Exec(`
	CREATE TABLE IF NOT EXISTS notice_sends (
		campaign TEXT NOT NULL,
		recipient TEXT NOT NULL,
		claimed_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
		resend_id TEXT NOT NULL DEFAULT '',
		PRIMARY KEY (campaign, recipient)
	)`)
	return err
}

func confirmedSubscribers(conn *sql.DB) ([]recipient, error) {
	rows, err := conn.Query(`
		SELECT id, email, unsubscribe_token FROM subscribers
		WHERE status = 'confirmed' ORDER BY id`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []recipient
	for rows.Next() {
		var id int64
		var r recipient
		if err := rows.Scan(&id, &r.Email, &r.Token); err != nil {
			return nil, err
		}
		r.Key = "subscriber:" + strconv.FormatInt(id, 10)
		out = append(out, r)
	}
	return out, rows.Err()
}

// buyerKey is a short hash of the normalised address, so the send log can
// tell buyers apart without storing addresses.
func buyerKey(email string) string {
	sum := sha256.Sum256([]byte(normalise(email)))
	return "buyer:" + hex.EncodeToString(sum[:8])
}

func normalise(email string) string { return strings.ToLower(strings.TrimSpace(email)) }

// withBuyers adds each buyer address that is not already a confirmed
// subscriber, once.
func withBuyers(subs []recipient, buyers []string) []recipient {
	seen := make(map[string]bool, len(subs)+len(buyers))
	out := append([]recipient(nil), subs...)
	for _, s := range subs {
		seen[normalise(s.Email)] = true
	}
	for _, b := range buyers {
		n := normalise(b)
		if n == "" || !strings.Contains(n, "@") || seen[n] {
			continue
		}
		seen[n] = true
		out = append(out, recipient{Key: buyerKey(n), Email: strings.TrimSpace(b)})
	}
	return out
}

// pending drops recipients already claimed for campaign.
func pending(conn *sql.DB, campaign string, all []recipient) ([]recipient, error) {
	rows, err := conn.Query(`SELECT recipient FROM notice_sends WHERE campaign = ?`, campaign)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	done := map[string]bool{}
	for rows.Next() {
		var k string
		if err := rows.Scan(&k); err != nil {
			return nil, err
		}
		done[k] = true
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	var out []recipient
	for _, r := range all {
		if !done[r.Key] {
			out = append(out, r)
		}
	}
	return out, nil
}

func claim(conn *sql.DB, campaign, key string) (bool, error) {
	res, err := conn.Exec(`INSERT OR IGNORE INTO notice_sends (campaign, recipient) VALUES (?, ?)`, campaign, key)
	if err != nil {
		return false, err
	}
	n, err := res.RowsAffected()
	return n == 1, err
}

func release(conn *sql.DB, campaign, key string) error {
	_, err := conn.Exec(`DELETE FROM notice_sends WHERE campaign = ? AND recipient = ?`, campaign, key)
	return err
}

func recordResendID(conn *sql.DB, campaign, key, resendID string) error {
	_, err := conn.Exec(`UPDATE notice_sends SET resend_id = ? WHERE campaign = ? AND recipient = ?`, resendID, campaign, key)
	return err
}

// --- Stripe buyers ----------------------------------------------------------

type stripeSession struct {
	ID              string            `json:"id"`
	Livemode        bool              `json:"livemode"`
	PaymentStatus   string            `json:"payment_status"`
	Metadata        map[string]string `json:"metadata"`
	CustomerDetails *struct {
		Email string `json:"email"`
	} `json:"customer_details"`
}

type sessionPage struct {
	Data    []stripeSession `json:"data"`
	HasMore bool            `json:"has_more"`
}

// boughtProduct reports whether a checkout session's metadata names code
// (Buy now sets "product", the cart sets "product_codes"), and whether the
// session carries a product code at all.
func boughtProduct(md map[string]string, code string) (bought, known bool) {
	if p, ok := md["product"]; ok {
		return p == code, true
	}
	if list, ok := md["product_codes"]; ok {
		for _, c := range strings.Split(list, ",") {
			if strings.TrimSpace(c) == code {
				return true, true
			}
		}
		return false, true
	}
	return false, false
}

// fetchBuyers lists the addresses of paid live-mode checkout sessions for
// code. unknown counts paid live sessions without a product code (skipped).
func fetchBuyers(client *http.Client, base, key, code string) (emails []string, unknown int, err error) {
	after := ""
	for page := 0; page < 200; page++ {
		q := url.Values{"status": {"complete"}, "limit": {"100"}}
		if after != "" {
			q.Set("starting_after", after)
		}
		req, err := http.NewRequest(http.MethodGet, base+"/v1/checkout/sessions?"+q.Encode(), nil)
		if err != nil {
			return nil, 0, err
		}
		req.Header.Set("Authorization", "Bearer "+key)
		resp, err := client.Do(req)
		if err != nil {
			return nil, 0, fmt.Errorf("stripe: %w", err)
		}
		body, err := io.ReadAll(io.LimitReader(resp.Body, 16<<20))
		resp.Body.Close()
		if err != nil {
			return nil, 0, fmt.Errorf("stripe: %w", err)
		}
		if resp.StatusCode != http.StatusOK {
			msg := string(body)
			if len(msg) > 300 {
				msg = msg[:300]
			}
			return nil, 0, fmt.Errorf("stripe: HTTP %d: %s", resp.StatusCode, msg)
		}
		var p sessionPage
		if err := json.Unmarshal(body, &p); err != nil {
			return nil, 0, fmt.Errorf("stripe: %w", err)
		}
		for _, s := range p.Data {
			if !s.Livemode || s.PaymentStatus != "paid" {
				continue
			}
			bought, known := boughtProduct(s.Metadata, code)
			if !known {
				unknown++
			}
			if !bought || s.CustomerDetails == nil || strings.TrimSpace(s.CustomerDetails.Email) == "" {
				continue
			}
			emails = append(emails, s.CustomerDetails.Email)
		}
		if !p.HasMore || len(p.Data) == 0 {
			return emails, unknown, nil
		}
		after = p.Data[len(p.Data)-1].ID
	}
	return nil, 0, errors.New("stripe: too many pages of checkout sessions")
}

// --- Messages ---------------------------------------------------------------

func render(tmpl, footer, unsubURL, address string, escape bool) string {
	if escape {
		unsubURL = html.EscapeString(unsubURL)
		address = html.EscapeString(address)
	}
	out := strings.Replace(tmpl, "{{FOOTER}}", footer, 1)
	return strings.NewReplacer("{{UNSUBSCRIBE_URL}}", unsubURL, "{{POSTAL_ADDRESS}}", address).Replace(out)
}

func buildMessage(baseURL string, r recipient, subject, address string) *resend.SendEmailRequest {
	req := &resend.SendEmailRequest{
		From:    fromAddr,
		To:      []string{r.Email},
		ReplyTo: replyAddr,
		Subject: subject,
	}
	if r.isSubscriber() {
		unsub := strings.TrimRight(baseURL, "/") + "/subscribe/unsubscribe?token=" + url.QueryEscape(r.Token)
		req.Html = render(releaseHTML, subscriberFooterHTML, unsub, address, true)
		req.Text = render(releaseText, subscriberFooterText, unsub, address, false)
		req.Headers = map[string]string{"List-Unsubscribe": "<" + unsub + ">"}
	} else {
		req.Html = render(releaseHTML, buyerFooterHTML, "", address, true)
		req.Text = render(releaseText, buyerFooterText, "", address, false)
	}
	return req
}

func sendOne(client *resend.Client, req *resend.SendEmailRequest) (string, error) {
	sent, err := client.Emails.SendWithContext(context.Background(), req)
	if err != nil {
		return "", fmt.Errorf("resend send failed: %w", err)
	}
	return sent.Id, nil
}

func main() {
	campaign := flag.String("campaign", "", "campaign id, e.g. popfocus-0.1.21-2026-11 (required)")
	address := flag.String("address", "", "postal address for the footer (required)")
	subject := flag.String("subject", defaultSubject, "email subject")
	testTo := flag.String("test-to", "", "send one subscriber and one buyer test email to this address and exit (no send log)")
	send := flag.Bool("send", false, "send to every pending recipient")
	buyers := flag.Bool("buyers", false, "also email everyone who paid for --product in Stripe live mode (needs STRIPE_SECRET_KEY)")
	product := flag.String("product", "POPF", "product code whose buyers --buyers emails")
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
	all, err := confirmedSubscribers(conn)
	if err != nil {
		log.Fatal("subscribers: ", err)
	}
	subs := len(all)
	if *buyers {
		key := os.Getenv("STRIPE_SECRET_KEY")
		if key == "" {
			log.Fatal("missing required env var: STRIPE_SECRET_KEY")
		}
		emails, unknown, err := fetchBuyers(&http.Client{Timeout: 30 * time.Second}, stripeAPI, key, *product)
		if err != nil {
			log.Fatal("buyers: ", err)
		}
		all = withBuyers(all, emails)
		log.Printf("stripe: %d paid live %s checkout(s); %d buyer(s) not on the list; %d paid session(s) without a product code skipped",
			len(emails), *product, len(all)-subs, unknown)
	}
	todo, err := pending(conn, *campaign, all)
	if err != nil {
		log.Fatal("recipients: ", err)
	}
	pendingSubs := 0
	for _, r := range todo {
		if r.isSubscriber() {
			pendingSubs++
		}
	}
	log.Printf("campaign %q: %d pending (%d subscribers, %d buyers not on the list)",
		*campaign, len(todo), pendingSubs, len(todo)-pendingSubs)

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
		for _, r := range []recipient{
			{Key: "test-subscriber", Email: *testTo, Token: "TEST"},
			{Key: "test-buyer", Email: *testTo},
		} {
			id, err := sendOne(client, buildMessage(baseURL, r, "[TEST] "+*subject, *address))
			if err != nil {
				log.Fatal(err)
			}
			log.Printf("%s test email accepted, resend id: %s", r.Key, id)
			time.Sleep(600 * time.Millisecond)
		}
	case *send:
		sent, failed := 0, 0
		for _, r := range todo {
			ok, err := claim(conn, *campaign, r.Key)
			if err != nil {
				log.Fatal("claim: ", err)
			}
			if !ok {
				continue
			}
			id, err := sendOne(client, buildMessage(baseURL, r, *subject, *address))
			if err != nil {
				if rerr := release(conn, *campaign, r.Key); rerr != nil {
					log.Printf("could not release claim for %s: %v", r.Key, rerr)
				}
				log.Printf("send failed for %s: %v", r.Key, err)
				failed++
				continue
			}
			if err := recordResendID(conn, *campaign, r.Key, id); err != nil {
				log.Printf("could not record resend id for %s: %v", r.Key, err)
			}
			log.Printf("sent to %s (resend %s)", r.Key, id)
			sent++
			time.Sleep(600 * time.Millisecond)
		}
		log.Printf("done: %d sent, %d failed", sent, failed)
	default:
		log.Println("dry run (no emails sent). Subject:", *subject)
		fmt.Println("----- text body preview (subscriber) -----")
		fmt.Print(render(releaseText, subscriberFooterText, baseURL+"/subscribe/unsubscribe?token=EXAMPLE", *address, false))
		fmt.Println("\n----- buyer footer instead -----")
		fmt.Println(render(buyerFooterText, "", "", *address, false))
	}
}
