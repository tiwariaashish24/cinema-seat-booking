package booking

import (
	"context"
	"sync"
	"testing"

	"github.com/redis/go-redis/v9"
)

func TestConcurrentBooking(t *testing.T) {
	rdb := redis.NewClient(&redis.Options{
		Addr: "localhost:6379",
	})
	defer rdb.Close()

	ctx := context.Background()
	if err := rdb.Ping(ctx).Err(); err != nil {
		t.Skip("Redis not available")
	}

	store := NewRedisStore(rdb)

	// Clean up test data
	testKey := "seat:test-movie:A1"
	rdb.Del(ctx, testKey)
	rdb.Del(ctx, lockKey("test-movie", "A1"))

	// Test concurrent booking attempts for the same seat
	numGoroutines := 10
	var wg sync.WaitGroup
	successCount := 0
	var mu sync.Mutex

	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(userID int) {
			defer wg.Done()

			booking := Booking{
				MovieID: "test-movie",
				SeatID:  "A1",
				UserID:  "user" + string(rune('0'+userID)),
			}

			err := store.Book(booking)
			if err == nil {
				mu.Lock()
				successCount++
				mu.Unlock()
			}
		}(i)
	}

	wg.Wait()

	// Only one booking should succeed
	if successCount != 1 {
		t.Errorf("Expected 1 successful booking, got %d", successCount)
	}

	// Clean up
	rdb.Del(ctx, testKey)
	rdb.Del(ctx, lockKey("test-movie", "A1"))
}

func TestDifferentSeatsConcurrent(t *testing.T) {
	rdb := redis.NewClient(&redis.Options{
		Addr: "localhost:6379",
	})
	defer rdb.Close()

	ctx := context.Background()
	if err := rdb.Ping(ctx).Err(); err != nil {
		t.Skip("Redis not available")
	}

	store := NewRedisStore(rdb)

	// Clean up test data
	for i := 0; i < 5; i++ {
		seatID := string(rune('A' + i))
		rdb.Del(ctx, "seat:test-movie:"+seatID)
		rdb.Del(ctx, lockKey("test-movie", seatID))
	}

	// Test concurrent booking attempts for different seats
	numGoroutines := 5
	var wg sync.WaitGroup
	successCount := 0
	var mu sync.Mutex

	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(seatIndex int) {
			defer wg.Done()

			seatID := string(rune('A' + seatIndex))
			booking := Booking{
				MovieID: "test-movie",
				SeatID:  seatID,
				UserID:  "user123",
			}

			err := store.Book(booking)
			if err == nil {
				mu.Lock()
				successCount++
				mu.Unlock()
			}
		}(i)
	}

	wg.Wait()

	// All bookings should succeed for different seats
	if successCount != numGoroutines {
		t.Errorf("Expected %d successful bookings, got %d", numGoroutines, successCount)
	}

	// Clean up
	for i := 0; i < 5; i++ {
		seatID := string(rune('A' + i))
		rdb.Del(ctx, "seat:test-movie:"+seatID)
		rdb.Del(ctx, lockKey("test-movie", seatID))
	}
}
