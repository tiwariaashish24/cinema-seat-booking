package booking

import (
	"time"
)

type MemoryStore struct {
	bookings map[string]Booking
}

func NewMemoryStore() *MemoryStore {
	return &MemoryStore{
		bookings: map[string]Booking{},
	}
}

func (s *MemoryStore) Book(b Booking) (Booking, error) {
	if _, exist := s.bookings[b.SeatID]; exist {
		return Booking{}, ErrorSeatAlreadyBooked
	}
	b.Status = "held"
	b.ExpiresAt = time.Now().Add(2 * time.Minute)
	s.bookings[b.SeatID] = b
	return b, nil
}

func (s *MemoryStore) ListBookings(MovieID string) []Booking {
	var result []Booking
	for _, b := range s.bookings {
		if b.MovieID == MovieID {
			result = append(result, b)
		}
	}
	return result
}

func (s *MemoryStore) ListMovies() []Movie {
	return []Movie{
		{ID: "1", Title: "The Matrix", Description: "A computer hacker learns about the true nature of reality.", Seats: 100},
		{ID: "2", Title: "Inception", Description: "A thief who steals corporate secrets through dream-sharing technology.", Seats: 120},
		{ID: "3", Title: "Interstellar", Description: "A team of explorers travel through a wormhole in space.", Seats: 80},
	}
}

func (s *MemoryStore) GetBooking(bookingID string) (Booking, error) {
	for _, b := range s.bookings {
		if b.ID == bookingID {
			return b, nil
		}
	}
	return Booking{}, ErrorBookingNotFound
}

func (s *MemoryStore) ConfirmBooking(bookingID string) error {
	b, err := s.GetBooking(bookingID)
	if err != nil {
		return err
	}

	if b.Status == "confirmed" {
		return ErrorBookingConfirmed
	}

	if b.Status == "held" && time.Now().After(b.ExpiresAt) {
		return ErrorBookingExpired
	}

	b.Status = "confirmed"
	b.ExpiresAt = time.Time{}
	s.bookings[b.SeatID] = b
	return nil
}

func (s *MemoryStore) CancelBooking(bookingID string) error {
	b, err := s.GetBooking(bookingID)
	if err != nil {
		return err
	}

	delete(s.bookings, b.SeatID)
	return nil
}

func (s *MemoryStore) ReleaseExpiredSeats() (int, error) {
	releasedCount := 0
	now := time.Now()

	for seatID, b := range s.bookings {
		if b.Status == "held" && now.After(b.ExpiresAt) {
			delete(s.bookings, seatID)
			releasedCount++
		}
	}

	return releasedCount, nil
}
