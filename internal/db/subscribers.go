package db

import (
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"errors"
)

// ErrAlreadySubscribed is returned when a confirmed subscriber tries to
// sign up again.
var ErrAlreadySubscribed = errors.New("already subscribed")

func newToken() (string, error) {
	b := make([]byte, 24)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}

// CreateSubscriber inserts a pending subscriber and returns the confirm
// token to email. If the email already exists and is confirmed, returns
// ErrAlreadySubscribed. If it exists but is pending or unsubscribed, the
// tokens are regenerated and status reset to pending (allows re-signup).
func CreateSubscriber(conn *sql.DB, email, source string) (confirmToken string, err error) {
	confirmToken, err = newToken()
	if err != nil {
		return "", err
	}
	unsubToken, err := newToken()
	if err != nil {
		return "", err
	}

	var existingStatus string
	err = conn.QueryRow(`SELECT status FROM subscribers WHERE email = ?`, email).Scan(&existingStatus)
	switch {
	case err == sql.ErrNoRows:
		_, err = conn.Exec(
			`INSERT INTO subscribers (email, status, confirm_token, unsubscribe_token, source) VALUES (?, 'pending', ?, ?, ?)`,
			email, confirmToken, unsubToken, source,
		)
		return confirmToken, err
	case err != nil:
		return "", err
	case existingStatus == "confirmed":
		return "", ErrAlreadySubscribed
	default:
		_, err = conn.Exec(
			`UPDATE subscribers SET status = 'pending', confirm_token = ?, unsubscribe_token = ?, source = ? WHERE email = ?`,
			confirmToken, unsubToken, source, email,
		)
		return confirmToken, err
	}
}

// CountConfirmedSubscribers returns the number of subscribers currently
// in 'confirmed' status.
func CountConfirmedSubscribers(conn *sql.DB) (int, error) {
	var count int
	err := conn.QueryRow(`SELECT COUNT(*) FROM subscribers WHERE status = 'confirmed'`).Scan(&count)
	return count, err
}

// ConfirmSubscriber marks a subscriber confirmed by their confirm token.
// Returns sql.ErrNoRows if the token doesn't match a pending subscriber.
func ConfirmSubscriber(conn *sql.DB, token string) error {
	res, err := conn.Exec(
		`UPDATE subscribers SET status = 'confirmed', confirmed_at = CURRENT_TIMESTAMP
		 WHERE confirm_token = ? AND status = 'pending'`,
		token,
	)
	if err != nil {
		return err
	}
	n, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if n == 0 {
		return sql.ErrNoRows
	}
	return nil
}

// UnsubscribeByToken marks a subscriber unsubscribed by their unsubscribe
// token. Returns sql.ErrNoRows if the token doesn't match anyone.
func UnsubscribeByToken(conn *sql.DB, token string) error {
	res, err := conn.Exec(
		`UPDATE subscribers SET status = 'unsubscribed' WHERE unsubscribe_token = ?`,
		token,
	)
	if err != nil {
		return err
	}
	n, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if n == 0 {
		return sql.ErrNoRows
	}
	return nil
}

// GetSubscriberByConfirmToken fetches the pending email for a confirm
// token, for use in the "confirm" handler.
func GetSubscriberByConfirmToken(conn *sql.DB, token string) (email, unsubToken string, err error) {
	err = conn.QueryRow(
		`SELECT email, unsubscribe_token FROM subscribers WHERE confirm_token = ?`,
		token,
	).Scan(&email, &unsubToken)
	return email, unsubToken, err
}
