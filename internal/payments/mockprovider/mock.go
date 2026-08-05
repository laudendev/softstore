package mockprovider

import "softstore/internal/payments"

// MockProvider is a test double for payments.Provider. It records every
// call it receives and returns canned or configurable responses, so tests
// can assert on behavior without touching a real payment API.
type MockProvider struct {
	RegisterItemCalls  []payments.SellableItem
	AddPriceCalls      []payments.AdditionalPrice
	StartPurchaseCalls []payments.PurchaseRequest

	// RegisterItemFunc, AddPriceFunc, and StartPurchaseFunc let a test
	// override behavior per-case (e.g. to simulate an error). If nil,
	// sensible defaults are used.
	RegisterItemFunc  func(payments.SellableItem) (payments.RegisteredItem, error)
	AddPriceFunc      func(payments.AdditionalPrice) (payments.RegisteredItem, error)
	StartPurchaseFunc func(payments.PurchaseRequest) (payments.Purchase, error)
}

func New() *MockProvider {
	return &MockProvider{}
}

func (m *MockProvider) RegisterItem(item payments.SellableItem) (payments.RegisteredItem, error) {
	m.RegisterItemCalls = append(m.RegisterItemCalls, item)
	if m.RegisterItemFunc != nil {
		return m.RegisterItemFunc(item)
	}
	return payments.RegisteredItem{ProviderItemID: "mock_item_id", ProviderProductID: "mock_product_id"}, nil
}

// AddPriceCalls and AddPriceFunc let tests inspect/override AddPrice the
// same way RegisterItem is handled above.
func (m *MockProvider) AddPrice(req payments.AdditionalPrice) (payments.RegisteredItem, error) {
	m.AddPriceCalls = append(m.AddPriceCalls, req)
	if m.AddPriceFunc != nil {
		return m.AddPriceFunc(req)
	}
	return payments.RegisteredItem{ProviderItemID: "mock_price_id_" + req.ProviderProductID, ProviderProductID: req.ProviderProductID}, nil
}

func (m *MockProvider) StartPurchase(req payments.PurchaseRequest) (payments.Purchase, error) {
	m.StartPurchaseCalls = append(m.StartPurchaseCalls, req)
	if m.StartPurchaseFunc != nil {
		return m.StartPurchaseFunc(req)
	}
	return payments.Purchase{RedirectURL: "https://mock-provider.test/checkout/session"}, nil
}
