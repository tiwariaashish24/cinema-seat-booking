package postgres

import (
	"context"
	"errors"
	"os"
	"testing"
	"time"

	"github.com/google/uuid"
)

func TestDemoBookingLifecycle(t *testing.T) {
	connectionString := os.Getenv("TEST_DATABASE_URL")
	if connectionString == "" {
		t.Skip("set TEST_DATABASE_URL to run PostgreSQL integration tests")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	store, err := Open(ctx, connectionString)
	if err != nil {
		t.Fatalf("open demo booking store: %v", err)
	}
	t.Cleanup(func() {
		if err := store.Close(); err != nil {
			t.Errorf("close demo booking store: %v", err)
		}
	})

	showtimes, err := store.ListShowtimes(ctx)
	if err != nil {
		t.Fatalf("list seeded showtimes: %v", err)
	}
	if len(showtimes) == 0 {
		t.Fatal("expected seeded demo showtimes")
	}

	var selectedShowtimeID string
	var selectedSeatID string
	for _, showtime := range showtimes {
		seats, err := store.ListSeats(ctx, showtime.ID)
		if err != nil {
			t.Fatalf("list seats for %s: %v", showtime.ID, err)
		}
		for _, seat := range seats {
			if seat.Status == "available" {
				selectedShowtimeID = showtime.ID
				selectedSeatID = seat.ID
				break
			}
		}
		if selectedSeatID != "" {
			break
		}
	}
	if selectedSeatID == "" {
		t.Fatal("no sample demo seat is available for the integration test")
	}

	customer := uuid.NewString()
	reservation, err := store.BookSeat(ctx, selectedShowtimeID, selectedSeatID, "tmdb-1", "Integration Movie", customer)
	if err != nil {
		t.Fatalf("reserve demo seat: %v", err)
	}
	t.Cleanup(func() {
		_ = store.CancelBooking(context.Background(), reservation.ID)
	})
	if !reservation.IsDemo || reservation.Status != "confirmed" {
		t.Fatalf("reservation = %#v, want a confirmed demo reservation", reservation)
	}

	_, err = store.BookSeat(ctx, selectedShowtimeID, selectedSeatID, "tmdb-2", "Another Movie", uuid.NewString())
	if !errors.Is(err, ErrSeatUnavailable) {
		t.Fatalf("booking occupied seat error = %v, want ErrSeatUnavailable", err)
	}

	retrieved, err := store.GetBooking(ctx, reservation.ID)
	if err != nil {
		t.Fatalf("get demo reservation: %v", err)
	}
	if retrieved.MovieTitle != "Integration Movie" || retrieved.Seat == "" {
		t.Fatalf("retrieved reservation = %#v, missing movie or seat", retrieved)
	}
	listed, err := store.ListBookings(ctx, customer)
	if err != nil {
		t.Fatalf("list demo reservations: %v", err)
	}
	if len(listed) != 1 || listed[0].ID != reservation.ID {
		t.Fatalf("listed reservations = %#v, want the new reservation", listed)
	}

	if err := store.CancelBooking(ctx, reservation.ID); err != nil {
		t.Fatalf("cancel demo reservation: %v", err)
	}
	seats, err := store.ListSeats(ctx, selectedShowtimeID)
	if err != nil {
		t.Fatalf("reload seats after cancellation: %v", err)
	}
	for _, seat := range seats {
		if seat.ID == selectedSeatID && seat.Status != "available" {
			t.Fatalf("cancelled seat status = %q, want available", seat.Status)
		}
	}
	if _, err := store.GetBooking(ctx, reservation.ID); err != nil {
		t.Fatalf("get cancelled reservation: %v", err)
	}
}
