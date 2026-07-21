package mockprovider

import "softstore/internal/payments"

// MockProvider is a test double for payments.Provider. It records every
// call it receives and returns canned or configurable responses, so tests
// can assert on behavior without touching a real payment API.
type MockProvider struct {
	RegisterItemCalls  []payments.SellableItem
	StartPurchaseCalls []payments.PurchaseRequest

	// RegisterItemFunc and StartPurchaseFunc let a test override behavior
	// per-case (e.g. to simulate an error). If nil, sensible defaults are used.
	RegisterItemFunc  func(payments.SellableItem) (payments.RegisteredItem, error)
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
	return payments.RegisteredItem{ProviderItemID: "mock_item_id"}, nil
}

func (m *MockProvider) StartPurchase(req payments.PurchaseRequest) (payments.Purchase, error) {
	m.StartPurchaseCalls = append(m.StartPurchaseCalls, req)
	if m.StartPurchaseFunc != nil {
		return m.StartPurchaseFunc(req)
	}
	return payments.Purchase{RedirectURL: "https://mock-provider.test/checkout/session"}, nil
}
