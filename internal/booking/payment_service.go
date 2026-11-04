package booking

import (
	"fmt"
	"booking/internal/adapter/razorpay"
)

type PaymentService struct {
	bookingService *Service
	paymentClient  razorpay.RazorpayClient
}

func NewPaymentService(bookingService *Service, paymentClient razorpay.RazorpayClient) *PaymentService {
	return &PaymentService{
		bookingService: bookingService,
		paymentClient:  paymentClient,
	}
}

func (s *PaymentService) CreatePaymentOrder(bookingID string, amount float64) (*razorpay.Order, error) {
	booking, err := s.bookingService.GetBooking(bookingID)
	if err != nil {
		return nil, err
	}

	if booking.Status != "held" {
		return nil, fmt.Errorf("booking is not in held status")
	}

	req := razorpay.CreateOrderRequest{
		Amount:   amount,
		Currency: "INR",
		Receipt:  bookingID,
		Notes: map[string]string{
			"booking_id": bookingID,
			"movie_id":   booking.MovieID,
			"seat_id":    booking.SeatID,
			"user_id":    booking.UserID,
		},
	}

	order, err := s.paymentClient.CreateOrder(req)
	if err != nil {
		return nil, fmt.Errorf("failed to create payment order: %w", err)
	}

	return order, nil
}

func (s *PaymentService) VerifyAndConfirmPayment(bookingID, orderID, paymentID, signature string) error {
	booking, err := s.bookingService.GetBooking(bookingID)
	if err != nil {
		return err
	}

	if booking.Status != "held" {
		return fmt.Errorf("booking is not in held status")
	}

	valid, err := s.paymentClient.VerifyPayment(orderID, paymentID, signature)
	if err != nil {
		return fmt.Errorf("payment verification failed: %w", err)
	}

	if !valid {
		return fmt.Errorf("invalid payment signature")
	}

	payment, err := s.paymentClient.GetPayment(paymentID)
	if err != nil {
		return fmt.Errorf("failed to get payment details: %w", err)
	}

	if payment.Status != "captured" {
		return fmt.Errorf("payment not captured")
	}

	if err := s.bookingService.ConfirmBooking(bookingID); err != nil {
		return fmt.Errorf("failed to confirm booking: %w", err)
	}

	return nil
}

func (s *PaymentService) IsTestMode() bool {
	return s.paymentClient.IsTestMode()
}
