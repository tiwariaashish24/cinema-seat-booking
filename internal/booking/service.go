package booking

type Service struct {
	store BookingStore
}

func NewService(store BookingStore) *Service {
	return &Service{store: store}
}

func (s *Service) Book(b Booking) (Booking, error) {
	return s.store.Book(b)
}

func (s *Service) ListMovies() []Movie {
	return s.store.ListMovies()
}

func (s *Service) GetBooking(bookingID string) (Booking, error) {
	return s.store.GetBooking(bookingID)
}

func (s *Service) ConfirmBooking(bookingID string) error {
	return s.store.ConfirmBooking(bookingID)
}

func (s *Service) CancelBooking(bookingID string) error {
	return s.store.CancelBooking(bookingID)
}

func (s *Service) ReleaseExpiredSeats() (int, error) {
	return s.store.ReleaseExpiredSeats()
}
