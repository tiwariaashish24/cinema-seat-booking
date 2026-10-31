package booking

type MemoryStore struct {
	bookings map[string]Booking
}

func NewMemoryStore() *MemoryStore {
	return &MemoryStore{
		bookings: map[string]Booking{},
	}
}

func (s *MemoryStore) Book(b Booking) error {
	if _, exist := s.bookings[b.SeatId]; exist {
		return ErrorSeatAlreadyBooked
	}
	s.bookings[b.SeatId] = b
	return nil
}

func (s *MemoryStore) ListBookings(MovieId string) []Booking {
	var result []Booking
	for _, b := range s.bookings {
		if b.MovieId == MovieId {
			result = append(result, b)
		}
	}
	return result
}
