package razorpay

import "time"

type Order struct {
	ID        string            `json:"id"`
	Amount    float64           `json:"amount"`
	Currency  string            `json:"currency"`
	Status    string            `json:"status"`
	Receipt   string            `json:"receipt"`
	Notes     map[string]string `json:"notes"`
	CreatedAt time.Time         `json:"created_at"`
}

type Payment struct {
	ID        string    `json:"id"`
	OrderID   string    `json:"order_id"`
	Amount    float64   `json:"amount"`
	Currency  string    `json:"currency"`
	Status    string    `json:"status"`
	Method    string    `json:"method"`
	Email     string    `json:"email"`
	Contact   string    `json:"contact"`
	CreatedAt time.Time `json:"created_at"`
}

type CreateOrderRequest struct {
	Amount   float64           `json:"amount"`
	Currency string            `json:"currency"`
	Receipt  string            `json:"receipt"`
	Notes    map[string]string `json:"notes"`
}

type VerifyPaymentRequest struct {
	OrderID   string `json:"order_id"`
	PaymentID string `json:"payment_id"`
	Signature string `json:"signature"`
}
