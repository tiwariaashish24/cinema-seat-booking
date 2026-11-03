package booking

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
)

func TestConcurrentBookingExactlyOneWins(t *testing.T) {
	rdb := redis.NewClient(&redis.Options{
		Addr:     "localhost:6379",
		PoolSize: 16,
	})
	t.Cleanup(func() {
		if err := rdb.Close(); err != nil {
			t.Errorf("close Redis client: %v", err)
		}
	})
	if err := rdb.Ping(t.Context()).Err(); err != nil {
		t.Fatalf("connect to Redis: %v", err)
	}

	store := NewRedisStore(rdb)
	svc := NewService(store)

	const numGoroutines = 1_000
	movieID := uuid.NewString()
	seatKey := "seat:" + movieID + ":A1"
	t.Cleanup(func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := rdb.Del(ctx, seatKey).Err(); err != nil {
			t.Errorf("clean up test seat: %v", err)
		}
	})

	var (
		successes atomic.Int64
		failures  atomic.Int64
		wg        sync.WaitGroup
	)
	unexpectedErrors := make(chan error, 1)

	wg.Add(numGoroutines)

	for range numGoroutines {
		go func() {
			defer wg.Done()

			err := svc.Book(Booking{
				MovieID: movieID,
				SeatID:  "A1",
				UserID:  uuid.New().String(),
			})

			if err == nil {
				successes.Add(1)
			} else if errors.Is(err, ErrorSeatAlreadyBooked) {
				failures.Add(1)
			} else {
				select {
				case unexpectedErrors <- err:
				default:
				}
			}
		}()
	}

	wg.Wait()
	select {
	case err := <-unexpectedErrors:
		t.Errorf("unexpected booking error: %v", err)
	default:
	}
	if got := successes.Load(); got != 1 {
		t.Errorf("expected exactly 1 success, got %d", got)
	}

	if got := failures.Load(); got != int64(numGoroutines-1) {
		t.Errorf("expected %d failures, got %d", numGoroutines-1, got)
	}
}
