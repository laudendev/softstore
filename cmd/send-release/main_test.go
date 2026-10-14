package main

import (
	"database/sql"
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

func addSub(t *testing.T, conn *sql.DB, email, status string) int64 {
	t.Helper()
	res, err := conn.Exec(`INSERT INTO subscribers (email, status, confirm_token, unsubscribe_token) VALUES (?, ?, ?, ?)`,
		email, status, "c-"+email, "u-"+email)
	if err != nil {
		t.Fatal(err)
	}
	id, _ := res.LastInsertId()
	return id
}

func TestPendingClaimAndRelease(t *testing.T) {
	conn := testConn(t)
	a := addSub(t, conn, "a@example.com", "confirmed")
	addSub(t, conn, "b@example.com", "confirmed")
	addSub(t, conn, "c@example.com", "pending")

	p, err := pendingRecipients(conn, "camp")
	if err != nil || len(p) != 2 {
		t.Fatalf("want 2 confirmed pending, got %d (%v)", len(p), err)
	}
	if ok, err := claim(conn, "camp", a); !ok || err != nil {
		t.Fatalf("first claim: %v %v", ok, err)
	}
	if ok, _ := claim(conn, "camp", a); ok {
		t.Fatal("second claim must fail")
	}
	if p, _ = pendingRecipients(conn, "camp"); len(p) != 1 {
		t.Fatalf("claimed subscriber must be excluded, got %d", len(p))
	}
	if p, _ = pendingRecipients(conn, "other"); len(p) != 2 {
		t.Fatalf("other campaign must be independent, got %d", len(p))
	}
	if err := release(conn, "camp", a); err != nil {
		t.Fatal(err)
	}
	if p, _ = pendingRecipients(conn, "camp"); len(p) != 2 {
		t.Fatalf("released claim must be retried, got %d", len(p))
	}
}

func TestBuildMessage(t *testing.T) {
	req := buildMessage("https://shop.example/", "x@example.com", "t k&1", "Subj", "1 A St & B, Town")
	unsub := "https://shop.example/subscribe/unsubscribe?token=t+k%261"
	if req.Headers["List-Unsubscribe"] != "<"+unsub+">" {
		t.Fatalf("header: %q", req.Headers["List-Unsubscribe"])
	}
	if !strings.Contains(req.Html, unsub) || !strings.Contains(req.Text, unsub) {
		t.Fatal("unsubscribe URL missing from a body")
	}
	if !strings.Contains(req.Html, "1 A St &amp; B, Town") || !strings.Contains(req.Text, "1 A St & B, Town") {
		t.Fatal("address missing or not escaped correctly")
	}
	if strings.Contains(req.Html, "{{") || strings.Contains(req.Text, "{{") {
		t.Fatal("unreplaced placeholder")
	}
	if req.From != fromAddr || len(req.To) != 1 || req.To[0] != "x@example.com" {
		t.Fatalf("addressing: %+v", req)
	}
}
