package booking

import "sync"

type ConcurrentStore struct {
	bookings map[string]Booking
	sync.RWMutex
}

func NewConcurrentStore() *ConcurrentStore {
	return &ConcurrentStore{
		bookings: map[string]Booking{},
	}
}

func (s *ConcurrentStore) Book(b Booking) error {
	s.Lock()
	defer s.Unlock()
	if _, exist := s.bookings[b.SeatId]; exist {
		return ErrorSeatAlreadyBooked
	}
	s.bookings[b.SeatId] = b
	return nil
}

func (s *ConcurrentStore) ListBookings(MovieId string) []Booking {
	s.RLock()
	defer s.RUnlock()
	var result []Booking
	for _, b := range s.bookings {
		if b.MovieId == MovieId {
			result = append(result, b)
		}
	}
	return result
}
