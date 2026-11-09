package postgres

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"booking/internal/cinema"

	"github.com/google/uuid"
	_ "github.com/jackc/pgx/v5/stdlib"
)

var (
	ErrShowtimeNotFound = errors.New("demo showtime not found")
	ErrSeatNotFound     = errors.New("demo seat not found")
	ErrSeatUnavailable  = errors.New("demo seat is not available")
	ErrBookingNotFound  = errors.New("demo booking not found")
)

type Store struct {
	db *sql.DB
}

func Open(ctx context.Context, connectionString string) (*Store, error) {
	db, err := sql.Open("pgx", connectionString)
	if err != nil {
		return nil, fmt.Errorf("open PostgreSQL connection: %w", err)
	}
	if err := db.PingContext(ctx); err != nil {
		db.Close()
		return nil, fmt.Errorf("connect to PostgreSQL: %w", err)
	}

	store := &Store{db: db}
	if err := store.migrate(ctx); err != nil {
		db.Close()
		return nil, err
	}
	return store, nil
}

func (s *Store) Close() error {
	return s.db.Close()
}

func (s *Store) migrate(ctx context.Context) error {
	const schema = `
CREATE TABLE IF NOT EXISTS demo_cinemas (
    id UUID PRIMARY KEY,
    name TEXT NOT NULL,
    address TEXT NOT NULL,
    is_demo BOOLEAN NOT NULL DEFAULT TRUE CHECK (is_demo)
);

CREATE TABLE IF NOT EXISTS demo_screens (
    id UUID PRIMARY KEY,
    cinema_id UUID NOT NULL REFERENCES demo_cinemas(id),
    name TEXT NOT NULL,
    UNIQUE (cinema_id, name)
);

CREATE TABLE IF NOT EXISTS demo_showtimes (
    id UUID PRIMARY KEY,
    screen_id UUID NOT NULL REFERENCES demo_screens(id),
    starts_at TIMESTAMPTZ NOT NULL,
    language TEXT NOT NULL,
    format TEXT NOT NULL,
    price_paise INTEGER NOT NULL CHECK (price_paise > 0),
    is_demo BOOLEAN NOT NULL DEFAULT TRUE CHECK (is_demo)
);

CREATE TABLE IF NOT EXISTS demo_seats (
    id UUID PRIMARY KEY,
    showtime_id UUID NOT NULL REFERENCES demo_showtimes(id),
    seat_number TEXT NOT NULL,
    status TEXT NOT NULL DEFAULT 'available' CHECK (status IN ('available', 'booked')),
    price_paise INTEGER NOT NULL CHECK (price_paise > 0),
    UNIQUE (showtime_id, seat_number),
    UNIQUE (id, showtime_id)
);

CREATE TABLE IF NOT EXISTS demo_bookings (
    id UUID PRIMARY KEY,
    showtime_id UUID NOT NULL,
    seat_id UUID NOT NULL,
    movie_id TEXT NOT NULL,
    movie_title TEXT NOT NULL,
    customer TEXT NOT NULL,
    status TEXT NOT NULL DEFAULT 'confirmed' CHECK (status IN ('confirmed', 'cancelled')),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    FOREIGN KEY (seat_id, showtime_id) REFERENCES demo_seats(id, showtime_id)
);

CREATE INDEX IF NOT EXISTS demo_bookings_customer_created_idx
    ON demo_bookings (customer, created_at DESC);

INSERT INTO demo_cinemas (id, name, address) VALUES
    ('10000000-0000-4000-8000-000000000001', 'SeatBook Sample Cinema — Connaught Place', 'Sample location: Connaught Place, New Delhi (not a real theatre)'),
    ('10000000-0000-4000-8000-000000000002', 'SeatBook Sample Cinema — Gurugram', 'Sample location: Sector 29, Gurugram (not a real theatre)')
ON CONFLICT (id) DO UPDATE SET name = EXCLUDED.name, address = EXCLUDED.address;

INSERT INTO demo_screens (id, cinema_id, name) VALUES
    ('20000000-0000-4000-8000-000000000001', '10000000-0000-4000-8000-000000000001', 'Sample Screen 1'),
    ('20000000-0000-4000-8000-000000000002', '10000000-0000-4000-8000-000000000002', 'Sample Screen 2')
ON CONFLICT (id) DO NOTHING;

INSERT INTO demo_showtimes (id, screen_id, starts_at, language, format, price_paise) VALUES
    ('30000000-0000-4000-8000-000000000001', '20000000-0000-4000-8000-000000000001', ((CURRENT_DATE + 1) + TIME '18:30') AT TIME ZONE 'Asia/Kolkata', 'Hindi', '2D', 25000),
    ('30000000-0000-4000-8000-000000000002', '20000000-0000-4000-8000-000000000002', ((CURRENT_DATE + 1) + TIME '21:30') AT TIME ZONE 'Asia/Kolkata', 'English', '2D', 30000)
ON CONFLICT (id) DO UPDATE SET screen_id = EXCLUDED.screen_id;

INSERT INTO demo_seats (id, showtime_id, seat_number, status, price_paise)
SELECT
    ('40000000-0000-4000-8000-' || LPAD((showtime_number * 100 + seat_number)::TEXT, 12, '0'))::UUID,
    CASE showtime_number
        WHEN 1 THEN '30000000-0000-4000-8000-000000000001'::UUID
        ELSE '30000000-0000-4000-8000-000000000002'::UUID
    END,
    CHR(64 + ((seat_number - 1) / 10 + 1)) || ((seat_number - 1) % 10 + 1)::TEXT,
    CASE WHEN seat_number IN (4, 17) THEN 'booked' ELSE 'available' END,
    CASE WHEN seat_number <= 20 THEN 30000 ELSE 25000 END
FROM generate_series(1, 2) AS showtime_number
CROSS JOIN generate_series(1, 40) AS seat_number
ON CONFLICT (id) DO NOTHING;
`
	if _, err := s.db.ExecContext(ctx, schema); err != nil {
		return fmt.Errorf("initialize demo cinema schema and seed data: %w", err)
	}
	return nil
}

func (s *Store) ListShowtimes(ctx context.Context) ([]cinema.DemoShowtime, error) {
	const query = `
SELECT st.id, c.name, c.address, sc.name, st.starts_at, st.language, st.format, st.price_paise,
       COUNT(seat.id) FILTER (WHERE seat.status = 'available'), st.is_demo
FROM demo_showtimes st
JOIN demo_screens sc ON sc.id = st.screen_id
JOIN demo_cinemas c ON c.id = sc.cinema_id
LEFT JOIN demo_seats seat ON seat.showtime_id = st.id
WHERE st.starts_at > NOW()
GROUP BY st.id, c.id, sc.id
ORDER BY st.starts_at`
	rows, err := s.db.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("list demo showtimes: %w", err)
	}
	defer rows.Close()

	showtimes := make([]cinema.DemoShowtime, 0)
	for rows.Next() {
		var showtime cinema.DemoShowtime
		if err := rows.Scan(
			&showtime.ID, &showtime.Cinema, &showtime.Address, &showtime.Screen,
			&showtime.StartsAt, &showtime.Language, &showtime.Format,
			&showtime.PricePaise, &showtime.Available, &showtime.IsDemo,
		); err != nil {
			return nil, fmt.Errorf("scan demo showtime: %w", err)
		}
		showtimes = append(showtimes, showtime)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate demo showtimes: %w", err)
	}
	return showtimes, nil
}

func (s *Store) ListSeats(ctx context.Context, showtimeID string) ([]cinema.DemoSeat, error) {
	if _, err := uuid.Parse(showtimeID); err != nil {
		return nil, ErrShowtimeNotFound
	}
	const query = `
SELECT id, seat_number, status, price_paise
FROM demo_seats
WHERE showtime_id = $1
ORDER BY substring(seat_number FROM '^[A-Z]+'), substring(seat_number FROM '[0-9]+')::INTEGER`
	rows, err := s.db.QueryContext(ctx, query, showtimeID)
	if err != nil {
		return nil, fmt.Errorf("list demo seats: %w", err)
	}
	defer rows.Close()

	seats := make([]cinema.DemoSeat, 0)
	for rows.Next() {
		var seat cinema.DemoSeat
		if err := rows.Scan(&seat.ID, &seat.Label, &seat.Status, &seat.PricePaise); err != nil {
			return nil, fmt.Errorf("scan demo seat: %w", err)
		}
		seats = append(seats, seat)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate demo seats: %w", err)
	}
	return seats, nil
}

func (s *Store) BookSeat(ctx context.Context, showtimeID, seatID, movieID, movieTitle, customer string) (cinema.DemoReservation, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return cinema.DemoReservation{}, fmt.Errorf("begin demo booking: %w", err)
	}
	defer tx.Rollback()

	const seatQuery = `
SELECT seat.status, seat.seat_number, seat.price_paise, c.name, sc.name, st.starts_at
FROM demo_seats seat
JOIN demo_showtimes st ON st.id = seat.showtime_id
JOIN demo_screens sc ON sc.id = st.screen_id
JOIN demo_cinemas c ON c.id = sc.cinema_id
WHERE st.id = $1 AND seat.id = $2 AND st.starts_at > NOW()
FOR UPDATE OF seat`
	var status, seatNumber string
	var seatPrice int
	var reservation cinema.DemoReservation
	if err := tx.QueryRowContext(ctx, seatQuery, showtimeID, seatID).
		Scan(&status, &seatNumber, &seatPrice, &reservation.Cinema, &reservation.Screen, &reservation.StartsAt); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return cinema.DemoReservation{}, ErrSeatNotFound
		}
		return cinema.DemoReservation{}, fmt.Errorf("read demo seat: %w", err)
	}
	if status != "available" {
		return cinema.DemoReservation{}, ErrSeatUnavailable
	}

	const insertQuery = `
INSERT INTO demo_bookings (id, showtime_id, seat_id, movie_id, movie_title, customer)
VALUES ($1, $2, $3, $4, $5, $6)
RETURNING id, created_at`
	reservationID := uuid.NewString()
	if err := tx.QueryRowContext(ctx, insertQuery, reservationID, showtimeID, seatID, movieID, movieTitle, customer).
		Scan(&reservation.ID, &reservation.CreatedAt); err != nil {
		return cinema.DemoReservation{}, fmt.Errorf("create demo booking: %w", err)
	}
	if _, err := tx.ExecContext(ctx, `UPDATE demo_seats SET status = 'booked' WHERE id = $1`, seatID); err != nil {
		return cinema.DemoReservation{}, fmt.Errorf("reserve demo seat: %w", err)
	}
	if err := tx.Commit(); err != nil {
		return cinema.DemoReservation{}, fmt.Errorf("commit demo booking: %w", err)
	}
	reservation.ShowtimeID = showtimeID
	reservation.MovieID = movieID
	reservation.MovieTitle = movieTitle
	reservation.Seat = seatNumber
	reservation.PricePaise = seatPrice
	reservation.Customer = customer
	reservation.Status = "confirmed"
	reservation.IsDemo = true
	return reservation, nil
}

func (s *Store) GetBooking(ctx context.Context, id string) (cinema.DemoReservation, error) {
	if _, err := uuid.Parse(id); err != nil {
		return cinema.DemoReservation{}, ErrBookingNotFound
	}
	const query = `
SELECT b.id, b.showtime_id, b.movie_id, b.movie_title, c.name, sc.name, seat.seat_number, seat.price_paise,
       b.customer, b.status, st.starts_at, b.created_at, st.is_demo
FROM demo_bookings b
JOIN demo_showtimes st ON st.id = b.showtime_id
JOIN demo_screens sc ON sc.id = st.screen_id
JOIN demo_cinemas c ON c.id = sc.cinema_id
JOIN demo_seats seat ON seat.id = b.seat_id
WHERE b.id = $1`
	var reservation cinema.DemoReservation
	err := s.db.QueryRowContext(ctx, query, id).Scan(
		&reservation.ID, &reservation.ShowtimeID, &reservation.MovieID, &reservation.MovieTitle,
		&reservation.Cinema, &reservation.Screen, &reservation.Seat, &reservation.PricePaise, &reservation.Customer,
		&reservation.Status, &reservation.StartsAt, &reservation.CreatedAt, &reservation.IsDemo,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return cinema.DemoReservation{}, ErrBookingNotFound
	}
	if err != nil {
		return cinema.DemoReservation{}, fmt.Errorf("get demo booking: %w", err)
	}
	return reservation, nil
}

func (s *Store) ListBookings(ctx context.Context, customer string) ([]cinema.DemoReservation, error) {
	const query = `
SELECT b.id, b.showtime_id, b.movie_id, b.movie_title, c.name, sc.name, seat.seat_number, seat.price_paise,
       b.customer, b.status, st.starts_at, b.created_at, st.is_demo
FROM demo_bookings b
JOIN demo_showtimes st ON st.id = b.showtime_id
JOIN demo_screens sc ON sc.id = st.screen_id
JOIN demo_cinemas c ON c.id = sc.cinema_id
JOIN demo_seats seat ON seat.id = b.seat_id
WHERE b.customer = $1
ORDER BY b.created_at DESC`
	rows, err := s.db.QueryContext(ctx, query, customer)
	if err != nil {
		return nil, fmt.Errorf("list demo bookings: %w", err)
	}
	defer rows.Close()

	reservations := make([]cinema.DemoReservation, 0)
	for rows.Next() {
		var reservation cinema.DemoReservation
		if err := rows.Scan(
			&reservation.ID, &reservation.ShowtimeID, &reservation.MovieID, &reservation.MovieTitle,
			&reservation.Cinema, &reservation.Screen, &reservation.Seat, &reservation.PricePaise, &reservation.Customer,
			&reservation.Status, &reservation.StartsAt, &reservation.CreatedAt, &reservation.IsDemo,
		); err != nil {
			return nil, fmt.Errorf("scan demo booking: %w", err)
		}
		reservations = append(reservations, reservation)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate demo bookings: %w", err)
	}
	return reservations, nil
}

func (s *Store) CancelBooking(ctx context.Context, id string) error {
	if _, err := uuid.Parse(id); err != nil {
		return ErrBookingNotFound
	}
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin demo booking cancellation: %w", err)
	}
	defer tx.Rollback()

	var seatID string
	err = tx.QueryRowContext(ctx, `SELECT seat_id FROM demo_bookings WHERE id = $1 AND status = 'confirmed' FOR UPDATE`, id).
		Scan(&seatID)
	if errors.Is(err, sql.ErrNoRows) {
		return ErrBookingNotFound
	}
	if err != nil {
		return fmt.Errorf("find demo booking to cancel: %w", err)
	}
	if _, err := tx.ExecContext(ctx, `UPDATE demo_bookings SET status = 'cancelled' WHERE id = $1`, id); err != nil {
		return fmt.Errorf("cancel demo booking: %w", err)
	}
	if _, err := tx.ExecContext(ctx, `UPDATE demo_seats SET status = 'available' WHERE id = $1`, seatID); err != nil {
		return fmt.Errorf("release demo seat: %w", err)
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit demo booking cancellation: %w", err)
	}
	return nil
}
