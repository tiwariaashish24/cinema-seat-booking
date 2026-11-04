package booking

import (
	"testing"
	"time"

	"booking/internal/adapter/razorpay"
)

func TestPaymentService_CreatePaymentOrder(t *testing.T) {
	store := NewMemoryStore()
	service := NewService(store)
	mockClient := razorpay.NewMockClient("test_key", "test_secret", true)
	paymentService := NewPaymentService(service, mockClient)

	booking := Booking{
		ID:        "booking-123",
		MovieID:   "movie-1",
		SeatID:    "A1",
		UserID:    "user-1",
		Status:    "held",
		ExpiresAt: time.Now().Add(5 * time.Minute),
	}

	_, err := store.Book(booking)
	if err != nil {
		t.Fatalf("Failed to create booking: %v", err)
	}

	order, err := paymentService.CreatePaymentOrder("booking-123", 500.00)
	if err != nil {
		t.Fatalf("CreatePaymentOrder failed: %v", err)
	}

	if order.ID == "" {
		t.Error("Expected order ID to be set")
	}

	if order.Amount != 500.00 {
		t.Errorf("Expected amount 500.00, got %f", order.Amount)
	}

	if order.Receipt != "booking-123" {
		t.Errorf("Expected receipt booking-123, got %s", order.Receipt)
	}

	_, err = paymentService.CreatePaymentOrder("nonexistent-booking", 500.00)
	if err == nil {
		t.Error("Expected error for nonexistent booking")
	}

	confirmedBooking := Booking{
		ID:        "booking-456",
		MovieID:   "movie-1",
		SeatID:    "A2",
		UserID:    "user-1",
		Status:    "held",
		ExpiresAt: time.Now().Add(5 * time.Minute),
	}

	_, err = store.Book(confirmedBooking)
	if err != nil {
		t.Fatalf("Failed to create booking: %v", err)
	}

	err = service.ConfirmBooking("booking-456")
	if err != nil {
		t.Fatalf("Failed to confirm booking: %v", err)
	}

	_, err = paymentService.CreatePaymentOrder("booking-456", 500.00)
	if err == nil {
		t.Error("Expected error for confirmed booking")
	}
}

func TestPaymentService_VerifyAndConfirmPayment(t *testing.T) {
	store := NewMemoryStore()
	service := NewService(store)
	mockClient := razorpay.NewMockClient("test_key", "test_secret", true)
	paymentService := NewPaymentService(service, mockClient)

	booking := Booking{
		ID:        "booking-123",
		MovieID:   "movie-1",
		SeatID:    "A1",
		UserID:    "user-1",
		Status:    "held",
		ExpiresAt: time.Now().Add(5 * time.Minute),
	}

	_, err := store.Book(booking)
	if err != nil {
		t.Fatalf("Failed to create booking: %v", err)
	}

	order, err := paymentService.CreatePaymentOrder("booking-123", 500.00)
	if err != nil {
		t.Fatalf("Failed to create payment order: %v", err)
	}

	payment := mockClient.CreatePayment(order.ID, "100")

	err = paymentService.VerifyAndConfirmPayment("booking-123", order.ID, payment.ID, "any_signature")
	if err != nil {
		t.Fatalf("VerifyAndConfirmPayment failed: %v", err)
	}

	retrievedBooking, err := service.GetBooking("booking-123")
	if err != nil {
		t.Fatalf("Failed to get booking: %v", err)
	}

	if retrievedBooking.Status != "confirmed" {
		t.Errorf("Expected booking status confirmed, got %s", retrievedBooking.Status)
	}

	err = paymentService.VerifyAndConfirmPayment("booking-123", order.ID, payment.ID, "any_signature")
	if err == nil {
		t.Error("Expected error when confirming already confirmed booking")
	}
}

func TestPaymentService_IsTestMode(t *testing.T) {
	store := NewMemoryStore()
	service := NewService(store)
	mockClient := razorpay.NewMockClient("test_key", "test_secret", true)
	paymentService := NewPaymentService(service, mockClient)

	if !paymentService.IsTestMode() {
		t.Error("Expected test mode to be true")
	}
}
