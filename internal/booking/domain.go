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

type Movie struct {
	ID          string
	Title       string
	Description string
	Seats       int
}

type BookingStore interface {
	Book(b Booking) error
	ListBookings(MovieID string) []Booking
	ListMovies() []Movie
}
