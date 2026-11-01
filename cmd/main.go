package main

import (
	"encoding/json"
	"log"
	"net/http"

	"booking/internal/adapter/redis"
	"booking/internal/booking"
)

func main() {
	rdb := redis.NewClient("localhost:6379")
	store := booking.NewRedisStore(rdb)
	service := booking.NewService(store)

	http.HandleFunc("/book", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		var b booking.Booking
		if err := json.NewDecoder(r.Body).Decode(&b); err != nil {
			http.Error(w, "Invalid request body", http.StatusBadRequest)
			return
		}

		if err := service.Book(b); err != nil {
			if err == booking.ErrorSeatAlreadyBooked {
				http.Error(w, err.Error(), http.StatusConflict)
				return
			}
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(b)
	})

	http.HandleFunc("/bookings/", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		movieID := r.URL.Path[len("/bookings/"):]
		bookings := store.ListBookings(movieID)

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(bookings)
	})

	log.Println("Server starting on :8082")
	log.Fatal(http.ListenAndServe(":8082", nil))
}
