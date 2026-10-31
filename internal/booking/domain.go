package booking

import "errors"

var (
	ErrorSeatAlreadyBooked = errors.New("The seat is already taken")
)

type Booking struct {
	Id      string
	MovieId string
	SeatId  string
	UserId  string
	Status  string
}

type BookingStore interface {
	Book(b Booking) error
	ListBookings(MovieId string) []Booking
}
