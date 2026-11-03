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

const defaultHoldTTL = 2 * time.Minute

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
	id := uuid.New().String()
	now := time.Now()

	ctx := context.Background()
	key := fmt.Sprintf("seat:%s:%s", b.MovieID, b.SeatID)

	b.ID = id
	val, err := json.Marshal(b)
	if err != nil {
		return Booking{}, fmt.Errorf("marshal booking: %w", err)
	}

	res := s.rdb.SetArgs(ctx, key, val, redis.SetArgs{
		Mode: "NX", // set if not exist
		TTL:  defaultHoldTTL,
	})
	if errors.Is(res.Err(), redis.Nil) {
		return Booking{}, ErrorSeatAlreadyBooked
	}
	if err := res.Err(); err != nil {
		return Booking{}, fmt.Errorf("reserve seat: %w", err)
	}

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
