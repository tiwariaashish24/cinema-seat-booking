package booking

import (
	"errors"
	"time"
)

var (
	ErrorSeatAlreadyBooked = errors.New("The seat is already taken")
)

type Booking struct {
	ID        string
	MovieID   string
	SeatID    string
	UserID    string
	Status    string
	ExpiresAt time.Time
}

type BookingStore interface {
	Book(b Booking) error
	ListBookings(MovieID string) []Booking
}
