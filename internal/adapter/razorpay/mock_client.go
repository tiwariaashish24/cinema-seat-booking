package razorpay

import (
	"fmt"
	"sync"
	"time"
)

type MockClient struct {
	orders    map[string]*Order
	payments  map[string]*Payment
	mu        sync.RWMutex
	apiKey    string
	apiSecret string
	testMode  bool
}

var _ RazorpayClient = (*MockClient)(nil)

func NewMockClient(apiKey, apiSecret string, testMode bool) *MockClient {
	return &MockClient{
		orders:    make(map[string]*Order),
		payments:  make(map[string]*Payment),
		apiKey:    apiKey,
		apiSecret: apiSecret,
		testMode:  testMode,
	}
}

func (c *MockClient) CreateOrder(req CreateOrderRequest) (*Order, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	orderID := fmt.Sprintf("order_%d", time.Now().UnixNano())
	order := &Order{
		ID:       orderID,
		Amount:   req.Amount,
		Currency: req.Currency,
		Status:   "created",
		Receipt:  req.Receipt,
		Notes:    req.Notes,
		CreatedAt: time.Now(),
	}

	c.orders[orderID] = order
	return order, nil
}

func (c *MockClient) VerifyPayment(orderID, paymentID, signature string) (bool, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	if _, ok := c.payments[paymentID]; !ok {
		return false, fmt.Errorf("payment not found")
	}

	if _, ok := c.orders[orderID]; !ok {
		return false, fmt.Errorf("order not found")
	}

	payment := c.payments[paymentID]
	if payment.OrderID != orderID {
		return false, fmt.Errorf("payment does not belong to this order")
	}

	if payment.Status != "captured" {
		return false, fmt.Errorf("payment not captured")
	}

	return true, nil
}

func (c *MockClient) GetPayment(paymentID string) (*Payment, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	payment, ok := c.payments[paymentID]
	if !ok {
		return nil, fmt.Errorf("payment not found")
	}

	return payment, nil
}

func (c *MockClient) CreatePayment(orderID, amount string) *Payment {
	c.mu.Lock()
	defer c.mu.Unlock()

	paymentID := fmt.Sprintf("pay_%d", time.Now().UnixNano())
	payment := &Payment{
		ID:        paymentID,
		OrderID:   orderID,
		Amount:    100.0,
		Currency:  "INR",
		Status:    "captured",
		Method:    "card",
		Email:     "test@example.com",
		Contact:   "9999999999",
		CreatedAt: time.Now(),
	}

	c.payments[paymentID] = payment
	return payment
}

func (c *MockClient) IsTestMode() bool {
	return c.testMode
}
