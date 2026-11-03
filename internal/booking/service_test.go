package booking

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
)

func TestConcurrentBookingExactlyOneWins(t *testing.T) {
	// Connect to Redis.
	rdb := redis.NewClient(&redis.Options{
		Addr:     "127.0.0.1:6379",
		PoolSize: 16,
	})

	t.Cleanup(func() {
		if err := rdb.Close(); err != nil {
			t.Errorf("close Redis client: %v", err)
		}
	})

	ctx := t.Context()

	if err := rdb.Ping(ctx).Err(); err != nil {
		t.Fatalf("connect to Redis: %v", err)
	}

	store := NewRedisStore(rdb)
	svc := NewService(store)

	const (
		numRequests = 100_000
		numWorkers  = 256
	)

	// Generate a unique movie and seat for this test.
	movieID := uuid.NewString()
	seatID := "A1"
	seatKey := "seat:" + movieID + ":" + seatID

	// Clean up the test seat using an independent context.
	t.Cleanup(func() {
		cleanupCtx, cancel := context.WithTimeout(
			context.Background(),
			5*time.Second,
		)
		defer cancel()

		if err := rdb.Del(cleanupCtx, seatKey).Err(); err != nil {
			t.Errorf("clean up test seat: %v", err)
		}
	})

	var (
		successes       atomic.Int64
		failures        atomic.Int64
		unexpectedCount atomic.Int64
		wg              sync.WaitGroup
	)

	unexpectedErrors := make(chan error, 1)
	requests := make(chan struct{}, numWorkers)

	start := time.Now()

	// Start bounded worker pool.
	wg.Add(numWorkers)

	for range numWorkers {
		go func() {
			defer wg.Done()

			for range requests {
				_, err := svc.Book(Booking{
					MovieID: movieID,
					SeatID:  seatID,
					UserID:  uuid.NewString(),
				})

				switch {
				case err == nil:
					successes.Add(1)

				case errors.Is(err, ErrorSeatAlreadyBooked):
					failures.Add(1)

				default:
					unexpectedCount.Add(1)

					select {
					case unexpectedErrors <- err:
					default:
					}
				}

			}
		}()
	}

	// Submit all 100,000 requests.
	for range numRequests {
		requests <- struct{}{}
	}

	close(requests)

	// Wait for every worker to finish.
	wg.Wait()

	elapsed := time.Since(start)
	totalSuccesses := successes.Load()
	totalFailures := failures.Load()
	totalUnexpected := unexpectedCount.Load()

	// Print performance metrics.
	rps := float64(numRequests) / elapsed.Seconds()

	fmt.Printf("\n========== BOOKING STRESS TEST ==========\n")
	fmt.Printf("Total Requests       : %d\n", numRequests)
	fmt.Printf("Workers              : %d\n", numWorkers)
	fmt.Printf("Successful Bookings  : %d\n", totalSuccesses)
	fmt.Printf("Booking Conflicts    : %d\n", totalFailures)
	fmt.Printf("Unexpected Errors    : %d\n", totalUnexpected)
	fmt.Printf("Execution Time       : %s\n", elapsed)
	fmt.Printf("Throughput           : %.2f requests/sec\n", rps)
	fmt.Printf("=========================================\n")

	// Report the first unexpected error.
	select {
	case err := <-unexpectedErrors:
		t.Errorf("unexpected booking error: %v", err)
	default:
	}

	// Verify exactly one booking succeeded.
	if totalSuccesses != 1 {
		t.Errorf(
			"expected exactly 1 successful booking, got %d",
			totalSuccesses,
		)
	}

	// Verify all remaining requests were booking conflicts.
	if totalFailures != numRequests-1 {
		t.Errorf(
			"expected %d booking conflicts, got %d",
			numRequests-1,
			totalFailures,
		)
	}

	// Verify no unexpected errors occurred.
	if totalUnexpected != 0 {
		t.Errorf(
			"expected 0 unexpected errors, got %d",
			totalUnexpected,
		)
	}
}

func TestConcurrentBookingDifferentSeats(t *testing.T) {
	const (
		numRequests = 10_000
		numWorkers  = 256
	)

	rdb := redis.NewClient(&redis.Options{
		Addr:     "127.0.0.1:6379",
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

	svc := NewService(NewRedisStore(rdb))
	movieID := uuid.NewString()
	seatKeys := make([]string, numRequests)
	for i := range numRequests {
		seatKeys[i] = fmt.Sprintf("seat:%s:A%d", movieID, i)
	}
	t.Cleanup(func() {
		cleanupCtx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
		defer cancel()
		if err := rdb.Del(cleanupCtx, seatKeys...).Err(); err != nil {
			t.Errorf("clean up test seats: %v", err)
		}
	})

	var (
		successes       atomic.Int64
		conflicts       atomic.Int64
		unexpectedCount atomic.Int64
		wg              sync.WaitGroup
	)
	unexpectedErrors := make(chan error, 1)
	requests := make(chan int, numWorkers)

	wg.Add(numWorkers)
	for range numWorkers {
		go func() {
			defer wg.Done()
			for seatNumber := range requests {
				_, err := svc.Book(Booking{
					MovieID: movieID,
					SeatID:  fmt.Sprintf("A%d", seatNumber),
					UserID:  uuid.NewString(),
				})
				switch {
				case err == nil:
					successes.Add(1)
				case errors.Is(err, ErrorSeatAlreadyBooked):
					conflicts.Add(1)
				default:
					unexpectedCount.Add(1)
					select {
					case unexpectedErrors <- err:
					default:
					}
				}
			}
		}()
	}

	for i := range numRequests {
		requests <- i
	}
	close(requests)
	wg.Wait()

	select {
	case err := <-unexpectedErrors:
		t.Errorf("unexpected booking error: %v", err)
	default:
	}
	if got := successes.Load(); got != numRequests {
		t.Errorf("expected %d successful bookings, got %d", numRequests, got)
	}
	if got := conflicts.Load(); got != 0 {
		t.Errorf("expected no booking conflicts, got %d", got)
	}
	if got := unexpectedCount.Load(); got != 0 {
		t.Errorf("expected no unexpected errors, got %d", got)
	}
}
