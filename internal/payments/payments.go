package payments

// SellableItem describes a product to register with a payment provider.
// PriceCents is the price in the smallest currency unit (cents for USD).
type SellableItem struct {
	Name        string
	Description string
	PriceCents  int64
	Currency    string // ISO 4217, e.g. "usd"
	TaxCategory string
}

// RegisteredItem is what a provider returns after successfully registering
// a SellableItem.
type RegisteredItem struct {
	// ProviderItemID is the provider's own identifier for this sellable item
	// (e.g. a Stripe Price ID). Opaque to softstore.
	ProviderItemID string
}

// LineItem is one entry in a checkout — a single sellable item and quantity.
type LineItem struct {
	ProviderItemID string
	Quantity       int64
}

// PurchaseRequest describes a checkout the customer should complete.
type PurchaseRequest struct {
	LineItems  []LineItem
	Metadata   map[string]string
	SuccessURL string
	CancelURL  string
}


// Purchase is what a provider returns after successfully starting a checkout.
type Purchase struct {
	RedirectURL string
}

// Provider is the seam softstore depends on for payment operations.
// Handlers depend only on this interface, never on a specific provider's SDK.
type Provider interface {
	RegisterItem(item SellableItem) (RegisteredItem, error)
	StartPurchase(req PurchaseRequest) (Purchase, error)
}
