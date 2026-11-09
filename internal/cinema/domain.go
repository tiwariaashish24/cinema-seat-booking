package cinema

import (
	"context"
	"time"
)

type Coordinates struct {
	Latitude  float64
	Longitude float64
}

type VenueProvider interface {
	Nearby(context.Context, string, *Coordinates) (NearbyCinemaResult, error)
}

type DemoBookingStore interface {
	ListShowtimes(context.Context) ([]DemoShowtime, error)
	ListSeats(context.Context, string) ([]DemoSeat, error)
	BookSeat(context.Context, string, string, string, string, string) (DemoReservation, error)
	GetBooking(context.Context, string) (DemoReservation, error)
	ListBookings(context.Context, string) ([]DemoReservation, error)
	CancelBooking(context.Context, string) error
}

type NearbyCinema struct {
	ID        string  `json:"id"`
	Name      string  `json:"name"`
	Address   string  `json:"address"`
	Latitude  float64 `json:"latitude"`
	Longitude float64 `json:"longitude"`
	Distance  float64 `json:"distance_km"`
	MapURL    string  `json:"map_url"`
}

type NearbyCinemaResult struct {
	Location    string         `json:"location"`
	Source      string         `json:"source"`
	Attribution string         `json:"attribution"`
	Cinemas     []NearbyCinema `json:"cinemas"`
}

type DemoShowtime struct {
	ID         string    `json:"id"`
	Cinema     string    `json:"cinema"`
	Address    string    `json:"address"`
	Screen     string    `json:"screen"`
	StartsAt   time.Time `json:"starts_at"`
	Language   string    `json:"language"`
	Format     string    `json:"format"`
	PricePaise int       `json:"price_paise"`
	Available  int       `json:"available_seats"`
	IsDemo     bool      `json:"is_demo"`
}

type DemoSeat struct {
	ID         string `json:"id"`
	Label      string `json:"label"`
	Status     string `json:"status"`
	PricePaise int    `json:"price_paise"`
}

type DemoReservation struct {
	ID         string    `json:"id"`
	ShowtimeID string    `json:"showtime_id"`
	MovieID    string    `json:"movie_id"`
	MovieTitle string    `json:"movie_title"`
	Cinema     string    `json:"cinema"`
	Screen     string    `json:"screen"`
	Seat       string    `json:"seat"`
	Customer   string    `json:"customer"`
	Status     string    `json:"status"`
	StartsAt   time.Time `json:"starts_at"`
	CreatedAt  time.Time `json:"created_at"`
	IsDemo     bool      `json:"is_demo"`
}
