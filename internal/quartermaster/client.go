package quartermaster

import (
	"encoding/json"
	"fmt"
	"net/http"
)

// SessionItem is one purchased item within a checkout session, as
// reported by Quartermaster.
type SessionItem struct {
	Product    string `json:"Product"`
	PriceID    string `json:"PriceID"`
	LicenseKey string `json:"LicenseKey"`
}

// SessionStatus reports whether a checkout session's license
// fulfillment has completed, and its items if so.
type SessionStatus struct {
	Found bool          `json:"found"`
	Ready bool          `json:"ready"`
	Items []SessionItem `json:"items"`
}

// Client calls Quartermaster's internal API over the WireGuard tunnel.
type Client struct {
	BaseURL       string // e.g. http://10.20.0.2:6774
	InternalSecret string
}

// GetSessionStatus polls Quartermaster for a checkout session's
// fulfillment status.
func (c *Client) GetSessionStatus(sessionID string) (SessionStatus, error) {
	url := fmt.Sprintf("%s/internal/sessions/%s/status", c.BaseURL, sessionID)
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return SessionStatus{}, fmt.Errorf("build request: %w", err)
	}
	req.Header.Set("X-Internal-Secret", c.InternalSecret)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return SessionStatus{}, fmt.Errorf("request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return SessionStatus{}, fmt.Errorf("unexpected status %d", resp.StatusCode)
	}

	var status SessionStatus
	if err := json.NewDecoder(resp.Body).Decode(&status); err != nil {
		return SessionStatus{}, fmt.Errorf("decode response: %w", err)
	}
	return status, nil
}
