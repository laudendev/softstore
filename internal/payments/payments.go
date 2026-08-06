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
	// ProviderProductID is the provider's identifier for the underlying
	// sellable product (e.g. a Stripe Product ID, distinct from the Price
	// ID above). Needed to attach additional prices to the same product
	// later — for example, a higher price for a multi-seat license tier
	// of the same underlying software.
	ProviderProductID string
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


// AdditionalPrice describes a new price to attach to an
// already-registered sellable item (e.g. a multi-seat tier of an
// existing product).
type AdditionalPrice struct {
	ProviderProductID string
	PriceCents        int64
	Currency          string
}

// Provider is the seam softstore depends on for payment operations.
// Handlers depend only on this interface, never on a specific provider's SDK.
type Provider interface {
	RegisterItem(item SellableItem) (RegisteredItem, error)
	AddPrice(req AdditionalPrice) (RegisteredItem, error)
	// UpdateProductDescription sets the customer-visible name/description
	// on an existing provider product. Checkout reads this live at
	// session-creation time, so calling this right before StartPurchase
	// lets a shared product's checkout page reflect which specific
	// price tier (e.g. "3 devices — 15% off") the buyer is purchasing,
	// without creating a new product per tier.
	UpdateProductDescription(providerProductID, name, description string) error
	StartPurchase(req PurchaseRequest) (Purchase, error)
}
