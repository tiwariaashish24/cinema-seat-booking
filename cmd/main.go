package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"

	"booking/internal/adapter/razorpay"
	"booking/internal/adapter/redis"
	"booking/internal/adapter/tmdb"
	"booking/internal/booking"

	"github.com/joho/godotenv"
)

func main() {
	if err := godotenv.Load(); err != nil {
		log.Println("Warning: .env file not found")
	}

	var store booking.BookingStore
	if os.Getenv("USE_MEMORY_STORE") == "true" || os.Getenv("USE_MEMORY_STORE") == "1" {
		log.Println("Using in-memory booking store (no Redis)")
		store = booking.NewMemoryStore()
	} else {
		rdb := redis.NewClient("localhost:6379")
		store = booking.NewRedisStore(rdb)
	}
	service := booking.NewService(store)
	tmdbClient := tmdb.NewClient()

	razorpayKey := os.Getenv("RAZORPAY_KEY_ID")
	razorpaySecret := os.Getenv("RAZORPAY_KEY_SECRET")
	testModeStr := os.Getenv("RAZORPAY_TEST_MODE")
	testMode := testModeStr == "true" || testModeStr == "1"

	var paymentClient razorpay.RazorpayClient
	var mockClient *razorpay.MockClient
	if razorpayKey == "" || razorpaySecret == "" {
		log.Println("Razorpay credentials not found, using mock client")
		mockClient = razorpay.NewMockClient("test_key", "test_secret", true)
		paymentClient = mockClient
	} else {
		paymentClient = razorpay.NewClient(razorpayKey, razorpaySecret, testMode)
	}

	paymentService := booking.NewPaymentService(service, paymentClient)

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

		bookingResult, err := service.Book(b)
		if err != nil {
			if err == booking.ErrorSeatAlreadyBooked {
				http.Error(w, err.Error(), http.StatusConflict)
				return
			}
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(bookingResult)
	})

	http.HandleFunc("/booking/", func(w http.ResponseWriter, r *http.Request) {
		bookingID := r.URL.Path[len("/booking/"):]
		if bookingID == "" {
			http.Error(w, "Booking ID is required", http.StatusBadRequest)
			return
		}

		switch r.Method {
		case http.MethodGet:
			b, err := service.GetBooking(bookingID)
			if err != nil {
				if err == booking.ErrorBookingNotFound {
					http.Error(w, err.Error(), http.StatusNotFound)
					return
				}
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}

			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(b)

		case http.MethodPost:
			// Confirm booking
			if err := service.ConfirmBooking(bookingID); err != nil {
				if err == booking.ErrorBookingNotFound {
					http.Error(w, err.Error(), http.StatusNotFound)
					return
				}
				if err == booking.ErrorBookingExpired {
					http.Error(w, err.Error(), http.StatusGone)
					return
				}
				if err == booking.ErrorBookingConfirmed {
					http.Error(w, err.Error(), http.StatusConflict)
					return
				}
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}

			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(map[string]string{"message": "Booking confirmed"})

		case http.MethodDelete:
			// Cancel booking
			if err := service.CancelBooking(bookingID); err != nil {
				if err == booking.ErrorBookingNotFound {
					http.Error(w, err.Error(), http.StatusNotFound)
					return
				}
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}

			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(map[string]string{"message": "Booking cancelled"})

		default:
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		}
	})

	http.HandleFunc("/admin/release-expired", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		count, err := service.ReleaseExpiredSeats()
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]int{"released": count})
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

	http.HandleFunc("/movies", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		movies := service.ListMovies()

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(movies)
	})

	http.HandleFunc("/api/movies/upcoming", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		page, err := parseMoviePage(r)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		movies, err := tmdbClient.GetUpcomingMovies(page)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(movies)
	})

	http.HandleFunc("/api/movies/popular", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		page, err := parseMoviePage(r)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		movies, err := tmdbClient.GetPopularMovies(page)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(movies)
	})

	http.HandleFunc("/api/movies/trending", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		page, err := parseMoviePage(r)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		movies, err := tmdbClient.GetTrendingMovies(page)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(movies)
	})

	http.HandleFunc("/api/movies/recommendations", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		movieID := r.URL.Query().Get("movie_id")
		if movieID == "" {
			http.Error(w, "movie_id query parameter is required", http.StatusBadRequest)
			return
		}

		var id int
		if _, err := fmt.Sscanf(movieID, "%d", &id); err != nil {
			http.Error(w, "invalid movie_id", http.StatusBadRequest)
			return
		}

		movies, err := tmdbClient.GetRecommendations(id)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(movies)
	})

	http.HandleFunc("/api/movies/search", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		query := r.URL.Query().Get("query")
		if query == "" {
			http.Error(w, "query parameter is required", http.StatusBadRequest)
			return
		}

		movies, err := tmdbClient.SearchMovies(query)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(movies)
	})

	http.HandleFunc("/payment/create-order", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		var req struct {
			BookingID string  `json:"booking_id"`
			Amount    float64 `json:"amount"`
		}

		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "Invalid request body", http.StatusBadRequest)
			return
		}

		if req.BookingID == "" {
			http.Error(w, "booking_id is required", http.StatusBadRequest)
			return
		}

		if req.Amount <= 0 {
			http.Error(w, "amount must be greater than 0", http.StatusBadRequest)
			return
		}

		order, err := paymentService.CreatePaymentOrder(req.BookingID, req.Amount)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(order)
	})

	http.HandleFunc("/payment/verify", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		var req struct {
			BookingID string `json:"booking_id"`
			OrderID   string `json:"order_id"`
			PaymentID string `json:"payment_id"`
			Signature string `json:"signature"`
		}

		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "Invalid request body", http.StatusBadRequest)
			return
		}

		if req.BookingID == "" || req.OrderID == "" || req.PaymentID == "" || req.Signature == "" {
			http.Error(w, "booking_id, order_id, payment_id, and signature are required", http.StatusBadRequest)
			return
		}

		if err := paymentService.VerifyAndConfirmPayment(req.BookingID, req.OrderID, req.PaymentID, req.Signature); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{"message": "Payment verified and booking confirmed"})
	})

	http.HandleFunc("/payment/webhook", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		var webhook struct {
			Event   string `json:"event"`
			Payload struct {
				Payment struct {
					ID      string            `json:"id"`
					OrderID string            `json:"order_id"`
					Status  string            `json:"status"`
					Amount  float64           `json:"amount"`
					Notes   map[string]string `json:"notes"`
				} `json:"payment"`
			} `json:"payload"`
		}

		if err := json.NewDecoder(r.Body).Decode(&webhook); err != nil {
			http.Error(w, "Invalid request body", http.StatusBadRequest)
			return
		}

		log.Printf("Received webhook: event=%s, payment_id=%s", webhook.Event, webhook.Payload.Payment.ID)

		if webhook.Event == "payment.captured" {
			bookingID := webhook.Payload.Payment.Notes["booking_id"]

			if webhook.Payload.Payment.Status == "captured" && bookingID != "" {
				if err := service.ConfirmBooking(bookingID); err != nil {
					log.Printf("Failed to confirm booking from webhook: %v", err)
				} else {
					log.Printf("Booking %s confirmed via webhook", bookingID)
				}
			}
		}

		w.WriteHeader(http.StatusOK)
	})

	http.HandleFunc("/payment/mock-pay", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		if mockClient == nil {
			http.Error(w, "mock payments are only available in test mode", http.StatusNotFound)
			return
		}

		var req struct {
			OrderID string  `json:"order_id"`
			Amount  float64 `json:"amount"`
		}

		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "Invalid request body", http.StatusBadRequest)
			return
		}

		if req.OrderID == "" {
			http.Error(w, "order_id is required", http.StatusBadRequest)
			return
		}

		payment := mockClient.CreatePayment(req.OrderID, fmt.Sprintf("%.2f", req.Amount))

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(payment)
	})

	http.HandleFunc("/payment/status", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"test_mode": paymentService.IsTestMode(),
		})
	})

	cors := func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Access-Control-Allow-Origin", "*")
			w.Header().Set("Access-Control-Allow-Methods", "GET, POST, DELETE, OPTIONS")
			w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
			if r.Method == http.MethodOptions {
				w.WriteHeader(http.StatusNoContent)
				return
			}
			next.ServeHTTP(w, r)
		})
	}

	log.Println("Server starting on :8082")
	log.Fatal(http.ListenAndServe(":8082", cors(http.DefaultServeMux)))
}

func parseMoviePage(r *http.Request) (int, error) {
	rawPage := r.URL.Query().Get("page")
	if rawPage == "" {
		return 1, nil
	}

	page, err := strconv.Atoi(rawPage)
	if err != nil || page < 1 || page > 500 {
		return 0, fmt.Errorf("page must be an integer between 1 and 500")
	}
	return page, nil
}
