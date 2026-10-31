package booking

import (
	"sync"
	"sync/atomic"
	"testing"
	"uuid"
)

func main() {

}

func TestConcurrentBookingExactlyOneWins(t *testing.T) {
	store := NewMemoryStore()
	svc := NewService(store)

	const nuwGoroutines = 100_00 ///100k user try to booking the seat at same time

	var (
		successes atomic.Int64
		failures  atomic.Int64
		wg        sync.WaitGroup
	)

	wg.Add(nuwGoroutines)
	for i := range nuwGoroutines {
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
	wg.Add()
}
