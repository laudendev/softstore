package quartermaster

import (
	"encoding/json"
	"fmt"
	"net/http"
)

// ReceiptItem is one purchased product's row in a checkout session's
// receipt, as reported by Quartermaster once fulfillment is complete.
type ReceiptItem struct {
	ProductName string `json:"product_name"`
	AmountLine  string `json:"amount_line"`
	LicenseKey  string `json:"license_key"`
}

// SessionStatus reports whether a checkout session's license
// fulfillment has completed, and its receipt detail if so.
type SessionStatus struct {
	Found     bool          `json:"found"`
	Ready     bool          `json:"ready"`
	Items     []ReceiptItem `json:"items"`
	TaxLine   string        `json:"tax_line"`
	TotalLine string        `json:"total_line"`
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
