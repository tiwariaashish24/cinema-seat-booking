package booking

import (
	"errors"
	"time"
)

var (
	ErrorSeatAlreadyBooked   = errors.New("The seat is already taken")
	ErrorBookingNotFound     = errors.New("Booking not found")
	ErrorBookingExpired      = errors.New("Booking has expired")
	ErrorBookingConfirmed    = errors.New("Booking is already confirmed")
	ErrorInvalidStatus       = errors.New("Invalid booking status")
)

type Booking struct {
	ID        string    `json:"id"`
	MovieID   string    `json:"movie_id"`
	SeatID    string    `json:"seat_id"`
	UserID    string    `json:"user_id"`
	Status    string    `json:"status"`
	ExpiresAt time.Time `json:"expires_at"`
}

type Movie struct {
	ID          string
	Title       string
	Description string
	Seats       int
}

type BookingStore interface {
	Book(b Booking) (Booking, error)
	ConfirmBooking(bookingID string) error
	CancelBooking(bookingID string) error
	ReleaseExpiredSeats() (int, error)
	GetBooking(bookingID string) (Booking, error)
	ListBookings(MovieID string) []Booking
	ListMovies() []Movie
}
