# Cinema Seat Booking System

A backend-focused cinema seat booking system built with **Go (Golang)** and **Redis**. This project explores concurrent seat reservation, temporary booking holds, booking lifecycle management, and distributed locking to prevent multiple users from reserving the same seat.

> **Project status:** Core booking APIs and Redis-backed seat reservation are implemented. Razorpay payment gateway integration has been added with test mode support.

## Table of Contents

- [Features](#features)
- [Tech Stack](#tech-stack)
- [Architecture](#architecture)
- [How Seat Booking Works](#how-seat-booking-works)
- [Project Structure](#project-structure)
- [Getting Started](#getting-started)
- [Environment Configuration](#environment-configuration)
- [API Reference](#api-reference)
- [Running Tests](#running-tests)
- [Concurrency and Load Testing](#concurrency-and-load-testing)
- [Roadmap](#roadmap)
- [Security Notes](#security-notes)
- [Contact](#contact)
- [License](#license)

## Features

- Create temporary seat reservations with an expiry time.
- Prevent duplicate reservations for the same movie and seat.
- Use Redis as the shared booking store.
- Use Redis-based distributed locks while performing seat operations.
- Confirm and cancel bookings.
- Retrieve a booking by its ID.
- List bookings for a movie.
- Release expired seat holds.
- Fetch upcoming, popular, trending, and recommended movies using TMDB.
- Search for movies.
- Provide an in-memory store implementation for development and tests.
- Include unit, lifecycle, Redis, and concurrency tests.
- **Razorpay payment gateway integration** with test mode support.
- **Payment order creation** for held bookings.
- **Payment verification** with signature validation.
- **Webhook support** for payment events.

## Tech Stack

| Technology  | Purpose                                              |
| ----------- | ---------------------------------------------------- |
| Go          | Backend language                                     |
| Redis       | Booking storage, seat holds, and distributed locking |
| go-redis    | Go Redis client                                      |
| Google UUID | Generate unique booking/session IDs                  |
| godotenv    | Load environment variables from `.env`               |
| TMDB API    | Movie discovery and search                           |
| Razorpay    | Payment gateway integration                          |
| Go testing  | Unit and concurrency testing                         |

## Architecture

```text
                         HTTP Client
                             |
                             v
                       cmd/main.go
                             |
                +------------+------------+
                |                         |
                v                         v
          Booking Service             TMDB Client
                |                         |
         +------+-------+                 v
         |              |              TMDB API
         v              v
     RedisStore      MemoryStore
         |
         v
       Redis
  (seat records, session
   mappings, and locks)
```

### Redis key design

The Redis store uses keys with the following patterns:

| Key                            | Purpose                                         |
| ------------------------------ | ----------------------------------------------- |
| `seat:{movieID}:{seatID}`      | Stores the booking data for a seat              |
| `session:{sessionID}`          | Maps a booking/session ID to its seat key       |
| `lock:seat:{movieID}:{seatID}` | Temporary distributed lock for a seat operation |

A new seat reservation is initially stored with a temporary hold. The current default hold duration is **two minutes**. A confirmed booking is stored without the temporary hold expiry.

## How Seat Booking Works

1. A client sends a booking request with a movie ID, seat ID, and user ID.
2. The backend attempts to acquire a short-lived Redis lock for that movie seat.
3. The backend checks whether the seat is already reserved.
4. If the seat is available, the backend creates a booking with `held` status.
5. The hold expires after the configured hold duration unless the booking is confirmed.
6. A valid held booking can be confirmed.
7. A booking can also be cancelled, and expired holds can be released.

```text
              Create booking
                    |
                    v
                   HELD
                 /     \
                /       \
          Confirm       Expire / Cancel
             |                |
             v                v
         CONFIRMED          RELEASED
```

> **Note:** Payment processing is not implemented yet. The current confirmation endpoint changes the booking status; it should not be treated as proof of a completed payment.

## Project Structure

```text
cinema-seat-booking/
├── cmd/
│   └── main.go
├── internal/
│   ├── adapter/
│   │   ├── redis/
│   │   └── tmdb/
│   └── booking/
│       ├── domain.go
│       ├── service.go
│       ├── memory_store.go
│       ├── redis_store.go
│       ├── service_test.go
│       ├── redis_store_test.go
│       └── lifecycle_test.go
├── .env                 # Local secrets; do not commit
├── .gitignore
├── go.mod
├── go.sum
├── LICENSE
└── README.md
```

The structure above describes the main application areas. Test files and supporting files may change as development continues.

## Getting Started

### Prerequisites

Install:

- Go version compatible with the project's `go.mod` file.
- Redis.
- Git.
- A TMDB API key if you want to use the movie endpoints.

### 1. Clone the repository

```bash
git clone https://github.com/tiwariaashish24/cinema-seat-booking.git
cd cinema-seat-booking
```

### 2. Download dependencies

```bash
go mod download
```

### 3. Start Redis

Start a local Redis server and make sure it is reachable at:

```text
localhost:6379
```

The current application entry point uses this Redis address.

### 4. Configure environment variables

Create a `.env` file in the project root using `.env.example` as a template:

```env
TMDB_TOKEN=your_tmdb_read_access_token
RAZORPAY_KEY_ID=your_razorpay_key_id
RAZORPAY_KEY_SECRET=your_razorpay_key_secret
RAZORPAY_TEST_MODE=true
```

If Razorpay credentials are not provided, the application will use a mock payment client for testing purposes.

Keep your `.env` file out of version control. Never commit API keys or other secrets.

### 5. Run the application

```bash
go run ./cmd
```

The HTTP server listens on:

```text
http://localhost:8082
```

## API Reference

The current server exposes the following endpoints:

| Method   | Endpoint                                    | Description                     |
| -------- | ------------------------------------------- | ------------------------------- |
| `POST`   | `/book`                                     | Create a temporary seat hold    |
| `GET`    | `/booking/{bookingID}`                      | Retrieve a booking              |
| `POST`   | `/booking/{bookingID}`                      | Confirm a booking               |
| `DELETE` | `/booking/{bookingID}`                      | Cancel a booking                |
| `POST`   | `/admin/release-expired`                    | Release expired seat holds      |
| `GET`    | `/bookings/{movieID}`                       | List bookings for a movie       |
| `GET`    | `/movies`                                   | List the sample movies          |
| `GET`    | `/api/movies/upcoming?page={page}`           | Get upcoming movies from TMDB   |
| `GET`    | `/api/movies/popular?page={page}`            | Get popular movies from TMDB    |
| `GET`    | `/api/movies/trending?page={page}`           | Get trending movies from TMDB   |
| `GET`    | `/api/movies/recommendations?movie_id={id}` | Get recommendations for a movie |
| `GET`    | `/api/movies/search?query={term}`           | Search movies                   |
| `POST`   | `/payment/create-order`                     | Create a payment order          |
| `POST`   | `/payment/verify`                           | Verify payment and confirm booking |
| `POST`   | `/payment/webhook`                          | Razorpay webhook handler        |
| `GET`    | `/payment/status`                           | Check payment gateway mode      |

### Create a booking

**Request**

```http
POST /book
Content-Type: application/json
```

**Request body**

```json
{
  "movie_id": "1",
  "seat_id": "A10",
  "user_id": "user-123"
}
```

The backend generates the booking ID and sets the booking status to `held`, along with an expiry time.

A successful request returns the booking details as JSON. If the seat is already reserved, the server returns `409 Conflict`.

### Retrieve a booking

```http
GET /booking/{bookingID}
```

### Confirm a booking

```http
POST /booking/{bookingID}
```

The current implementation confirms a valid held booking. It returns a conflict if the booking is already confirmed and a gone response if the hold has expired.

### Cancel a booking

```http
DELETE /booking/{bookingID}
```

### Release expired holds

```http
POST /admin/release-expired
```

This endpoint releases expired held bookings. It is an administrative endpoint and should be protected before public deployment.

### List bookings for a movie

```http
GET /bookings/{movieID}
```

### Movie endpoints

```http
GET /movies
GET /api/movies/upcoming
GET /api/movies/popular
GET /api/movies/trending
GET /api/movies/recommendations?movie_id=1
GET /api/movies/search?query=inception
```

TMDB endpoints require a TMDB API Read Access Token in `TMDB_TOKEN`. The movie lists return 20 results per page; `page` defaults to 1.

### Payment endpoints

#### Create payment order

```http
POST /payment/create-order
Content-Type: application/json
```

**Request body**

```json
{
  "booking_id": "booking-123",
  "amount": 500.00
}
```

Creates a Razorpay order for a held booking. The amount should be in the smallest currency unit (e.g., paise for INR).

#### Verify payment

```http
POST /payment/verify
Content-Type: application/json
```

**Request body**

```json
{
  "booking_id": "booking-123",
  "order_id": "order_abc123",
  "payment_id": "pay_xyz789",
  "signature": "generated_signature"
}
```

Verifies the payment signature and confirms the booking if the payment is successful.

#### Payment webhook

```http
POST /payment/webhook
Content-Type: application/json
```

Handles Razorpay webhook events. Currently processes `payment.captured` events to confirm bookings automatically.

#### Payment status

```http
GET /payment/status
```

Returns the current payment gateway mode (test or production).

## Running Tests

Run all tests:

```bash
go test ./... -count=1
```

Run tests with Go's race detector:

```bash
go test -race ./... -count=1
```

Redis-backed tests need a reachable Redis instance and any test configuration expected by the test suite.

## Concurrency and Load Testing

The project includes concurrent booking tests that check whether multiple requests can reserve the same seat at the same time. The expected behavior is that only one request succeeds for a particular seat.

During local experiments, the test workload was increased as high as **100,000 goroutines**. On Windows, extreme simultaneous Redis connection activity caused socket buffer exhaustion. The workload was reduced to **1,000 concurrent attempts** for stable test runs and race-detector verification.

These results describe local testing conditions. They do not represent a benchmark proving that the service can handle 100,000 production requests. Further performance work should include:

- Bounded worker pools instead of creating an unrestricted number of concurrent workers.
- Redis connection-pool configuration and reuse.
- Repeatable load tests with controlled request rates.
- Measurements of latency, throughput, and error rates.
- Testing on the intended deployment environment.

## Roadmap

- [x] Integrate Razorpay in test mode.
- [x] Create payment orders from the Go backend.
- [x] Verify payment signatures on the server.
- [x] Process payment webhooks safely and idempotently.
- [x] Confirm bookings only after verified payment.
- [ ] Add authentication and authorization.
- [ ] Move Redis connection settings into environment configuration.
- [ ] Add graceful server shutdown and request timeouts.
- [ ] Improve expired-hold cleanup and background processing.
- [ ] Add repeatable load benchmarks and API documentation.
- [ ] Add webhook signature verification for Razorpay webhooks.

## Security Notes

- Do not commit `.env` files, API keys, passwords, or other secrets.
- Validate incoming request data.
- Add authentication and authorization before exposing booking and administrative endpoints publicly.
- Protect `/admin/release-expired` from unauthorized access.
- When payment integration is added, verify payment signatures on the backend and do not trust client-side payment status alone.

## Contact

For queries, feedback, or collaboration, reach out:

- **LinkedIn:** [Ashish Kumar Tiwari](https://www.linkedin.com/in/ashish-kumar-tiwari-812698296/)
- **Email:** [tiwariiashish04@gmail.com](mailto:tiwariiashish04@gmail.com)
- **GitHub:** [github.com/tiwariaashish24](https://github.com/tiwariaashish24)

## License

This project is licensed under the **MIT License**. See the [LICENSE](LICENSE) file for details.

Copyright (c) 2026 Ashish Kumar Tiwari
