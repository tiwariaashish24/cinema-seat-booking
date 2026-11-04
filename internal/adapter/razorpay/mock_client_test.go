package razorpay

import (
	"testing"
)

func TestMockClient_CreateOrder(t *testing.T) {
	client := NewMockClient("test_key", "test_secret", true)

	req := CreateOrderRequest{
		Amount:   500.00,
		Currency: "INR",
		Receipt:  "booking-123",
		Notes: map[string]string{
			"booking_id": "booking-123",
		},
	}

	order, err := client.CreateOrder(req)
	if err != nil {
		t.Fatalf("CreateOrder failed: %v", err)
	}

	if order.ID == "" {
		t.Error("Expected order ID to be set")
	}

	if order.Amount != 500.00 {
		t.Errorf("Expected amount 500.00, got %f", order.Amount)
	}

	if order.Currency != "INR" {
		t.Errorf("Expected currency INR, got %s", order.Currency)
	}

	if order.Status != "created" {
		t.Errorf("Expected status created, got %s", order.Status)
	}

	if order.Receipt != "booking-123" {
		t.Errorf("Expected receipt booking-123, got %s", order.Receipt)
	}
}

func TestMockClient_CreatePayment(t *testing.T) {
	client := NewMockClient("test_key", "test_secret", true)

	orderID := "order_123"
	payment := client.CreatePayment(orderID, "100")

	if payment.ID == "" {
		t.Error("Expected payment ID to be set")
	}

	if payment.OrderID != orderID {
		t.Errorf("Expected order ID %s, got %s", orderID, payment.OrderID)
	}

	if payment.Status != "captured" {
		t.Errorf("Expected status captured, got %s", payment.Status)
	}
}

func TestMockClient_VerifyPayment(t *testing.T) {
	client := NewMockClient("test_key", "test_secret", true)

	req := CreateOrderRequest{
		Amount:   500.00,
		Currency: "INR",
		Receipt:  "booking-123",
	}

	order, _ := client.CreateOrder(req)
	orderID := order.ID
	payment := client.CreatePayment(orderID, "100")

	valid, err := client.VerifyPayment(orderID, payment.ID, "any_signature")
	if err != nil {
		t.Fatalf("VerifyPayment failed: %v", err)
	}

	if !valid {
		t.Error("Expected payment to be valid")
	}

	_, err = client.VerifyPayment("nonexistent_order", payment.ID, "any_signature")
	if err == nil {
		t.Error("Expected error for nonexistent order")
	}

	_, err = client.VerifyPayment(orderID, "nonexistent_payment", "any_signature")
	if err == nil {
		t.Error("Expected error for nonexistent payment")
	}
}

func TestMockClient_GetPayment(t *testing.T) {
	client := NewMockClient("test_key", "test_secret", true)

	req := CreateOrderRequest{
		Amount:   500.00,
		Currency: "INR",
		Receipt:  "booking-123",
	}

	order, _ := client.CreateOrder(req)
	orderID := order.ID
	payment := client.CreatePayment(orderID, "100")

	retrievedPayment, err := client.GetPayment(payment.ID)
	if err != nil {
		t.Fatalf("GetPayment failed: %v", err)
	}

	if retrievedPayment.ID != payment.ID {
		t.Errorf("Expected payment ID %s, got %s", payment.ID, retrievedPayment.ID)
	}

	_, err = client.GetPayment("nonexistent_payment")
	if err == nil {
		t.Error("Expected error for nonexistent payment")
	}
}

func TestMockClient_IsTestMode(t *testing.T) {
	client := NewMockClient("test_key", "test_secret", true)
	if !client.IsTestMode() {
		t.Error("Expected test mode to be true")
	}

	client = NewMockClient("test_key", "test_secret", false)
	if client.IsTestMode() {
		t.Error("Expected test mode to be false")
	}
}
