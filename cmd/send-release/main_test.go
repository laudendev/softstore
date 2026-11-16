package main

import (
	"database/sql"
	"fmt"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"

	"softstore/internal/db"
)

func testConn(t *testing.T) *sql.DB {
	t.Helper()
	conn, err := db.Open(filepath.Join(t.TempDir(), "t.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { conn.Close() })
	if err := ensureSendLog(conn); err != nil {
		t.Fatal(err)
	}
	return conn
}

func addSub(t *testing.T, conn *sql.DB, email, status string) {
	t.Helper()
	_, err := conn.Exec(`INSERT INTO subscribers (email, status, confirm_token, unsubscribe_token) VALUES (?, ?, ?, ?)`,
		email, status, "c-"+email, "u-"+email)
	if err != nil {
		t.Fatal(err)
	}
}

func TestPendingClaimAndRelease(t *testing.T) {
	conn := testConn(t)
	addSub(t, conn, "a@example.com", "confirmed")
	addSub(t, conn, "b@example.com", "confirmed")
	addSub(t, conn, "c@example.com", "pending")

	subs, err := confirmedSubscribers(conn)
	if err != nil || len(subs) != 2 {
		t.Fatalf("want 2 confirmed, got %d (%v)", len(subs), err)
	}
	all := withBuyers(subs, []string{"buyer@example.com"})
	a, buyer := all[0], all[2]
	p, err := pending(conn, "camp", all)
	if err != nil || len(p) != 3 {
		t.Fatalf("want 3 pending, got %d (%v)", len(p), err)
	}
	for _, key := range []string{a.Key, buyer.Key} {
		if ok, err := claim(conn, "camp", key); !ok || err != nil {
			t.Fatalf("first claim of %s: %v %v", key, ok, err)
		}
		if ok, _ := claim(conn, "camp", key); ok {
			t.Fatalf("second claim of %s must fail", key)
		}
	}
	if p, _ = pending(conn, "camp", all); len(p) != 1 {
		t.Fatalf("claimed recipients must be excluded, got %d", len(p))
	}
	if p, _ = pending(conn, "other", all); len(p) != 3 {
		t.Fatalf("other campaign must be independent, got %d", len(p))
	}
	if err := release(conn, "camp", buyer.Key); err != nil {
		t.Fatal(err)
	}
	if p, _ = pending(conn, "camp", all); len(p) != 2 {
		t.Fatalf("released claim must be retried, got %d", len(p))
	}
	var stored string
	if err := conn.QueryRow(`SELECT group_concat(recipient) FROM notice_sends`).Scan(&stored); err != nil {
		t.Fatal(err)
	}
	if strings.Contains(stored, "@") {
		t.Fatalf("send log must not hold addresses: %q", stored)
	}
}

func TestWithBuyersMergesAndDedupes(t *testing.T) {
	subs := []recipient{{Key: "subscriber:1", Email: "Sub@Example.com", Token: "u"}}
	all := withBuyers(subs, []string{
		"sub@example.com", // already a subscriber: subscriber version only
		"one@example.com",
		" ONE@example.com ", // same buyer twice
		"",
		"not-an-address",
		"two@example.com",
	})
	if len(all) != 3 {
		t.Fatalf("want subscriber + 2 buyers, got %+v", all)
	}
	if !all[0].isSubscriber() || all[1].isSubscriber() || all[2].isSubscriber() {
		t.Fatalf("kinds: %+v", all)
	}
	if all[1].Key != buyerKey("one@example.com") || all[1].Key == all[2].Key {
		t.Fatalf("keys: %q %q", all[1].Key, all[2].Key)
	}
	if strings.Contains(all[1].Key, "@") || !strings.HasPrefix(all[1].Key, "buyer:") {
		t.Fatalf("key must not hold the address: %q", all[1].Key)
	}
}

func TestBuildMessageForSubscriber(t *testing.T) {
	r := recipient{Key: "subscriber:1", Email: "x@example.com", Token: "t k&1"}
	req := buildMessage("https://shop.example/", r, "Subj", "1 A St & B, Town")
	unsub := "https://shop.example/subscribe/unsubscribe?token=t+k%261"
	if req.Headers["List-Unsubscribe"] != "<"+unsub+">" {
		t.Fatalf("header: %q", req.Headers["List-Unsubscribe"])
	}
	if !strings.Contains(req.Html, escapeAmp(unsub)) || !strings.Contains(req.Text, unsub) {
		t.Fatal("unsubscribe URL missing from a body")
	}
	if !strings.Contains(req.Text, "you subscribed to release updates") {
		t.Fatal("subscriber footer missing")
	}
	checkCommon(t, req.Html, req.Text)
	if req.From != fromAddr || len(req.To) != 1 || req.To[0] != "x@example.com" {
		t.Fatalf("addressing: %+v", req)
	}
}

func TestBuildMessageForBuyer(t *testing.T) {
	r := recipient{Key: buyerKey("y@example.com"), Email: "y@example.com"}
	req := buildMessage("https://shop.example/", r, "Subj", "1 A St & B, Town")
	if len(req.Headers) != 0 {
		t.Fatalf("a buyer notice has no List-Unsubscribe header: %v", req.Headers)
	}
	if strings.Contains(req.Html, "nsubscribe") || strings.Contains(req.Text, "nsubscribe") {
		t.Fatal("a buyer notice has no unsubscribe link")
	}
	for _, body := range []string{req.Html, req.Text} {
		if !strings.Contains(body, "you bought a Pop Focus") || !strings.Contains(body, "not on our mailing list") {
			t.Fatal("buyer footer missing")
		}
	}
	checkCommon(t, req.Html, req.Text)
}

func checkCommon(t *testing.T, htmlBody, text string) {
	t.Helper()
	if !strings.Contains(htmlBody, "1 A St &amp; B, Town") || !strings.Contains(text, "1 A St & B, Town") {
		t.Fatal("address missing or not escaped correctly")
	}
	if strings.Contains(htmlBody, "{{") || strings.Contains(text, "{{") {
		t.Fatal("unreplaced placeholder")
	}
	for _, body := range []string{htmlBody, text} {
		if !strings.Contains(body, "0.1.21") || !strings.Contains(body, "https://popfocus.lauden.dev/download/") {
			t.Fatal("notice content missing")
		}
	}
}

func escapeAmp(s string) string { return strings.ReplaceAll(s, "&", "&amp;") }

func TestFetchBuyers(t *testing.T) {
	pages := []string{
		`{"has_more": true, "data": [
			{"id": "cs_1", "livemode": true, "payment_status": "paid", "metadata": {"product": "POPF"}, "customer_details": {"email": "buy@example.com"}},
			{"id": "cs_2", "livemode": true, "payment_status": "paid", "metadata": {"product": "BOOK"}, "customer_details": {"email": "book@example.com"}},
			{"id": "cs_3", "livemode": false, "payment_status": "paid", "metadata": {"product": "POPF"}, "customer_details": {"email": "test@example.com"}},
			{"id": "cs_4", "livemode": true, "payment_status": "unpaid", "metadata": {"product": "POPF"}, "customer_details": {"email": "unpaid@example.com"}}
		]}`,
		`{"has_more": false, "data": [
			{"id": "cs_5", "livemode": true, "payment_status": "paid", "metadata": {"product_codes": "BOOK, POPF"}, "customer_details": {"email": "cart@example.com"}},
			{"id": "cs_6", "livemode": true, "payment_status": "paid", "metadata": {}, "customer_details": {"email": "old@example.com"}},
			{"id": "cs_7", "livemode": true, "payment_status": "paid", "metadata": {"product": "POPF"}, "customer_details": null}
		]}`,
	}
	calls := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Bearer rk_live_x" {
			http.Error(w, "no auth", http.StatusUnauthorized)
			return
		}
		if r.URL.Path != "/v1/checkout/sessions" || r.URL.Query().Get("status") != "complete" {
			http.Error(w, "bad request "+r.URL.String(), http.StatusBadRequest)
			return
		}
		want := ""
		if calls == 1 {
			want = "cs_4"
		}
		if got := r.URL.Query().Get("starting_after"); got != want {
			http.Error(w, fmt.Sprintf("starting_after %q, want %q", got, want), http.StatusBadRequest)
			return
		}
		fmt.Fprint(w, pages[calls])
		calls++
	}))
	defer srv.Close()

	emails, unknown, err := fetchBuyers(srv.Client(), srv.URL, "rk_live_x", "POPF")
	if err != nil {
		t.Fatal(err)
	}
	if strings.Join(emails, ",") != "buy@example.com,cart@example.com" {
		t.Fatalf("emails: %v", emails)
	}
	if unknown != 1 || calls != 2 {
		t.Fatalf("unknown %d calls %d", unknown, calls)
	}

	if _, _, err := fetchBuyers(srv.Client(), srv.URL, "wrong", "POPF"); err == nil || !strings.Contains(err.Error(), "401") {
		t.Fatalf("want HTTP 401 error, got %v", err)
	}
}
