package booking

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"time"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
)

const (
	defaultHoldTTL    = 2 * time.Minute
	lockTTL          = 10 * time.Second
	lockRetryDelay   = 50 * time.Millisecond
	maxLockRetries   = 20
)

// RedisStore implements session-based seat booking backed by Redis.
//
// Key design:
//
// seat:{movieID}:{seatID}  -> session JSON (TTL = held, no TTL = confirmed)
// session:{sessionID}     -> seat key (reverse lookup)

type RedisStore struct {
	rdb *redis.Client
}

func NewRedisStore(rdb *redis.Client) *RedisStore {
	return &RedisStore{rdb: rdb}
}

// sessionKey builds the reverse-lookup key for a session.
func sessionKey(id string) string {
	return fmt.Sprintf("session:%s", id)
}

// lockKey builds the distributed lock key for a seat.
func lockKey(movieID, seatID string) string {
	return fmt.Sprintf("lock:seat:%s:%s", movieID, seatID)
}

// acquireLock attempts to acquire a distributed lock for a seat using Redis SET NX.
func (s *RedisStore) acquireLock(ctx context.Context, movieID, seatID string) (bool, error) {
	key := lockKey(movieID, seatID)
	lockValue := uuid.New().String()

	for i := 0; i < maxLockRetries; i++ {
		acquired, err := s.rdb.SetNX(ctx, key, lockValue, lockTTL).Result()
		if err != nil {
			return false, fmt.Errorf("acquire lock: %w", err)
		}

		if acquired {
			return true, nil
		}

		// Lock is held by another process, wait and retry
		if i < maxLockRetries-1 {
			time.Sleep(lockRetryDelay)
		}
	}

	return false, ErrorSeatAlreadyBooked
}

// releaseLock releases the distributed lock for a seat.
func (s *RedisStore) releaseLock(ctx context.Context, movieID, seatID string) error {
	key := lockKey(movieID, seatID)
	if err := s.rdb.Del(ctx, key).Err(); err != nil {
		return fmt.Errorf("release lock: %w", err)
	}
	return nil
}

func (s *RedisStore) Book(b Booking) error {
	session, err := s.hold(b)
	if err != nil {
		return err
	}

	log.Printf("Session booked %v", session)

	return nil
}

func (s *RedisStore) ListBookings(movieID string) []Booking {
	return []Booking{}
}

func (s *RedisStore) ListMovies() []Movie {
	return []Movie{
		{ID: "1", Title: "The Matrix", Description: "A computer hacker learns about the true nature of reality.", Seats: 100},
		{ID: "2", Title: "Inception", Description: "A thief who steals corporate secrets through dream-sharing technology.", Seats: 120},
		{ID: "3", Title: "Interstellar", Description: "A team of explorers travel through a wormhole in space.", Seats: 80},
	}
}

func (s *RedisStore) hold(b Booking) (Booking, error) {
	ctx := context.Background()

	// Acquire distributed lock for this seat
	acquired, err := s.acquireLock(ctx, b.MovieID, b.SeatID)
	if err != nil {
		return Booking{}, err
	}
	if !acquired {
		return Booking{}, ErrorSeatAlreadyBooked
	}

	// Ensure lock is released when function exits
	defer func() {
		if releaseErr := s.releaseLock(ctx, b.MovieID, b.SeatID); releaseErr != nil {
			log.Printf("Failed to release lock for seat %s:%s: %v", b.MovieID, b.SeatID, releaseErr)
		}
	}()

	id := uuid.New().String()
	now := time.Now()
	key := fmt.Sprintf("seat:%s:%s", b.MovieID, b.SeatID)

	b.ID = id
	val, err := json.Marshal(b)
	if err != nil {
		return Booking{}, fmt.Errorf("marshal booking: %w", err)
	}

	// Check if seat is already booked
	exists, err := s.rdb.Exists(ctx, key).Result()
	if err != nil {
		return Booking{}, fmt.Errorf("check seat existence: %w", err)
	}
	if exists > 0 {
		return Booking{}, ErrorSeatAlreadyBooked
	}

	// Book the seat
	res := s.rdb.SetArgs(ctx, key, val, redis.SetArgs{
		Mode: "NX",
		TTL:  defaultHoldTTL,
	})
	if errors.Is(res.Err(), redis.Nil) {
		return Booking{}, ErrorSeatAlreadyBooked
	}
	if err := res.Err(); err != nil {
		return Booking{}, fmt.Errorf("reserve seat: %w", err)
	}

	// Store session mapping
	if err := s.rdb.Set(ctx, sessionKey(id), key, defaultHoldTTL).Err(); err != nil {
		if rollbackErr := s.rdb.Del(ctx, key).Err(); rollbackErr != nil {
			return Booking{}, fmt.Errorf("save booking session: %w (release reserved seat: %v)", err, rollbackErr)
		}
		return Booking{}, fmt.Errorf("save booking session: %w", err)
	}

	return Booking{
		ID:        id,
		MovieID:   b.MovieID,
		SeatID:    b.SeatID,
		UserID:    b.UserID,
		Status:    "held",
		ExpiresAt: now.Add(defaultHoldTTL),
	}, nil
}
