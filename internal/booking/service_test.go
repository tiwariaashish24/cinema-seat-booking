package booking

import (
	"sync"
	"sync/atomic"
	"testing"

	"github.com/google/uuid"
)

func TestConcurrentBookingExactlyOneWins(t *testing.T) {
	store := NewConcurrentStore()
	svc := NewService(store)

	const numGoroutines = 10_000

	var (
		successes atomic.Int64
		failures  atomic.Int64
		wg        sync.WaitGroup
	)

	wg.Add(numGoroutines)

	for i := range numGoroutines {
		go func(userNum int) {
			defer wg.Done()

			err := svc.Book(Booking{
				MovieId: "screen-1",
				SeatId:  "A1",
				UserId:  uuid.New().String(),
			})

			if err == nil {
				successes.Add(1)
			} else {
				failures.Add(1)
			}
		}(i)
	}

	wg.Wait()
	if got := successes.Load(); got != 1 {
		t.Errorf("expected exactly 1 success, got %d", got)
	}

	if got := failures.Load(); got != int64(numGoroutines-1) {
		t.Errorf("expected %d failures, got %d", numGoroutines-1, got)
	}
}
