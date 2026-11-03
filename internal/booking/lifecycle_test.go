package booking

import (
	"context"
	"encoding/json"
	"fmt"
	"testing"
	"time"

	"github.com/redis/go-redis/v9"
)

func TestBookingLifecycle(t *testing.T) {
	rdb := redis.NewClient(&redis.Options{
		Addr: "localhost:6379",
	})
	defer rdb.Close()

	ctx := context.Background()
	if err := rdb.Ping(ctx).Err(); err != nil {
		t.Skip("Redis not available")
	}

	store := NewRedisStore(rdb)

	// Create a booking
	req := Booking{
		MovieID: "test-movie",
		SeatID:  "A1",
		UserID:  "user123",
	}

	b, err := store.Book(req)
	if err != nil {
		t.Fatalf("Failed to create booking: %v", err)
	}
	bookingID := b.ID

	if bookingID == "" {
		t.Fatal("Booking ID is empty")
	}

	// Test GetBooking
	booking, err := store.GetBooking(bookingID)
	if err != nil {
		t.Fatalf("Failed to get booking: %v", err)
	}
	if booking.Status != "held" {
		t.Errorf("Expected status 'held', got '%s'", booking.Status)
	}

	// Test ConfirmBooking
	err = store.ConfirmBooking(bookingID)
	if err != nil {
		t.Fatalf("Failed to confirm booking: %v", err)
	}

	// Verify status changed
	confirmedBooking, err := store.GetBooking(bookingID)
	if err != nil {
		t.Fatalf("Failed to get booking after confirmation: %v", err)
	}
	if confirmedBooking.Status != "confirmed" {
		t.Errorf("Expected status 'confirmed', got '%s'", confirmedBooking.Status)
	}

	// Test double confirmation
	err = store.ConfirmBooking(bookingID)
	if err != ErrorBookingConfirmed {
		t.Errorf("Expected ErrorBookingConfirmed, got %v", err)
	}

	// Clean up
	testKey := "seat:test-movie:A1"
	rdb.Del(ctx, testKey)
	rdb.Del(ctx, sessionKey(bookingID))
	rdb.Del(ctx, lockKey("test-movie", "A1"))
}

func TestCancelBooking(t *testing.T) {
	rdb := redis.NewClient(&redis.Options{
		Addr: "localhost:6379",
	})
	defer rdb.Close()

	ctx := context.Background()
	if err := rdb.Ping(ctx).Err(); err != nil {
		t.Skip("Redis not available")
	}

	store := NewRedisStore(rdb)

	// Create a booking
	req := Booking{
		MovieID: "test-movie",
		SeatID:  "B1",
		UserID:  "user456",
	}

	b, err := store.Book(req)
	if err != nil {
		t.Fatalf("Failed to create booking: %v", err)
	}
	bookingID := b.ID

	// Cancel the booking
	err = store.CancelBooking(bookingID)
	if err != nil {
		t.Fatalf("Failed to cancel booking: %v", err)
	}

	// Verify booking is deleted
	_, err = store.GetBooking(bookingID)
	if err != ErrorBookingNotFound {
		t.Errorf("Expected ErrorBookingNotFound, got %v", err)
	}

	// Clean up
	rdb.Del(ctx, "seat:test-movie:B1")
	rdb.Del(ctx, sessionKey(bookingID))
	rdb.Del(ctx, lockKey("test-movie", "B1"))
}

func TestReleaseExpiredSeats(t *testing.T) {
	rdb := redis.NewClient(&redis.Options{
		Addr: "localhost:6379",
	})
	defer rdb.Close()

	ctx := context.Background()
	if err := rdb.Ping(ctx).Err(); err != nil {
		t.Skip("Redis not available")
	}

	store := NewRedisStore(rdb)

	// Create a booking with short TTL
	req := Booking{
		MovieID: "test-movie",
		SeatID:  "C1",
		UserID:  "user789",
	}

	booked, err := store.Book(req)
	if err != nil {
		t.Fatalf("Failed to create booking: %v", err)
	}
	bookingID := booked.ID

	// Manually set expiration to past
	key := fmt.Sprintf("seat:%s:%s", req.MovieID, req.SeatID)
	val, _ := rdb.Get(ctx, key).Result()
	var b Booking
	json.Unmarshal([]byte(val), &b)
	b.ExpiresAt = time.Now().Add(-1 * time.Hour)
	newVal, _ := json.Marshal(b)
	rdb.Set(ctx, key, newVal, 0)

	// Release expired seats
	count, err := store.ReleaseExpiredSeats()
	if err != nil {
		t.Fatalf("Failed to release expired seats: %v", err)
	}

	if count != 1 {
		t.Errorf("Expected to release 1 seat, got %d", count)
	}

	// Verify booking is deleted
	_, err = store.GetBooking(bookingID)
	if err != ErrorBookingNotFound {
		t.Errorf("Expected ErrorBookingNotFound, got %v", err)
	}

	// Clean up
	rdb.Del(ctx, "seat:test-movie:C1")
	rdb.Del(ctx, sessionKey(bookingID))
	rdb.Del(ctx, lockKey("test-movie", "C1"))
}
