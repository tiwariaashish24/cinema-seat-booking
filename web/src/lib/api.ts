const API = process.env.NEXT_PUBLIC_API_URL ?? "http://localhost:8082";

export interface Booking {
  id: string;
  movie_id: string;
  seat_id: string;
  user_id: string;
  status: "held" | "confirmed" | string;
  expires_at: string;
}

// Sample movies from GET /movies use Go's default JSON keys (capitalized).
export interface SampleMovie {
  ID?: string;
  id?: string;
  Title?: string;
  title?: string;
  Description?: string;
  description?: string;
  Seats?: number;
  seats?: number;
}

export interface TmdbMovie {
  id: number;
  title: string;
  overview: string;
  release_date: string;
  poster_path: string | null;
  vote_average: number;
}

export interface NearbyCinema {
  id: string;
  name: string;
  address: string;
  latitude: number;
  longitude: number;
  distance_km: number;
  map_url: string;
}

export interface NearbyCinemaResult {
  location: string;
  source: string;
  attribution: string;
  cinemas: NearbyCinema[];
}

export interface DemoShowtime {
  id: string;
  cinema: string;
  address: string;
  screen: string;
  starts_at: string;
  language: string;
  format: string;
  price_paise: number;
  available_seats: number;
  is_demo: boolean;
}

export interface DemoSeat {
  id: string;
  label: string;
  status: "available" | "booked";
  price_paise: number;
}

export interface DemoReservation {
  id: string;
  showtime_id: string;
  movie_id: string;
  movie_title: string;
  cinema: string;
  screen: string;
  seat: string;
  customer: string;
  status: "confirmed" | "cancelled";
  starts_at: string;
  created_at: string;
  is_demo: boolean;
}

export interface Order {
  id: string;
  amount: number;
  currency: string;
  status: string;
  receipt: string;
  notes: Record<string, string>;
  created_at: string;
}

export interface Payment {
  id: string;
  order_id: string;
  amount: number;
  currency: string;
  status: string;
}

async function req<T>(path: string, init?: RequestInit): Promise<T> {
  const res = await fetch(`${API}${path}`, {
    ...init,
    headers: { "Content-Type": "application/json", ...init?.headers },
  });
  if (!res.ok) {
    const text = await res.text();
    throw new Error(text || `Request failed: ${res.status}`);
  }
  return res.json() as Promise<T>;
}

const post = <T,>(path: string, body: unknown) =>
  req<T>(path, { method: "POST", body: JSON.stringify(body) });

export const api = {
  listMovies: () => req<SampleMovie[]>("/movies"),
  listBookings: (movieId: string) => req<Booking[]>(`/bookings/${movieId}`),
  createBooking: (movieId: string, seatId: string, userId: string) =>
    post<Booking>("/book", { movie_id: movieId, seat_id: seatId, user_id: userId }),
  getBooking: (id: string) => req<Booking>(`/booking/${id}`),
  confirmBooking: (id: string) =>
    post<{ message: string }>(`/booking/${id}`, {}),
  cancelBooking: (id: string) =>
    req<{ message: string }>(`/booking/${id}`, { method: "DELETE" }),
  releaseExpired: () => post<{ released: number }>("/admin/release-expired", {}),

  upcoming: (page = 1) => req<TmdbMovie[]>(`/api/movies/upcoming?page=${page}`),
  popular: (page = 1) => req<TmdbMovie[]>(`/api/movies/popular?page=${page}`),
  trending: (page = 1) => req<TmdbMovie[]>(`/api/movies/trending?page=${page}`),
  nowPlaying: (page = 1) => req<TmdbMovie[]>(`/api/movies/now-playing?page=${page}`),
  newReleases: (page = 1) => req<TmdbMovie[]>(`/api/movies/new-releases?page=${page}`),
  search: (query: string) =>
    req<TmdbMovie[]>(`/api/movies/search?query=${encodeURIComponent(query)}`),
  nearbyCinemas: (location: string) =>
    req<NearbyCinemaResult>(`/api/cinemas/nearby?location=${encodeURIComponent(location)}`),
  nearbyCinemasAt: (latitude: number, longitude: number) =>
    req<NearbyCinemaResult>(`/api/cinemas/nearby?lat=${latitude}&lon=${longitude}`),
  demoShowtimes: () => req<DemoShowtime[]>("/api/demo/showtimes"),
  demoSeats: (showtimeId: string) =>
    req<DemoSeat[]>(`/api/demo/showtimes/${encodeURIComponent(showtimeId)}/seats`),
  createDemoBooking: (booking: {
    showtime_id: string;
    seat_id: string;
    movie_id: string;
    movie_title: string;
    customer: string;
  }) => post<DemoReservation>("/api/demo/bookings/", booking),
  getDemoBooking: (id: string) =>
    req<DemoReservation>(`/api/demo/bookings/${encodeURIComponent(id)}`),
  listDemoBookings: (customer: string) =>
    req<DemoReservation[]>(`/api/demo/bookings?customer=${encodeURIComponent(customer)}`),
  cancelDemoBooking: (id: string) =>
    req<void>(`/api/demo/bookings/${encodeURIComponent(id)}`, { method: "DELETE" }),

  createOrder: (bookingId: string, amount: number) =>
    post<Order>("/payment/create-order", { booking_id: bookingId, amount }),
  verifyPayment: (b: {
    booking_id: string;
    order_id: string;
    payment_id: string;
    signature: string;
  }) => post<{ message: string }>("/payment/verify", b),
  mockPay: (orderId: string) =>
    post<Payment>("/payment/mock-pay", { order_id: orderId }),
  paymentStatus: () => req<{ test_mode: boolean }>("/payment/status"),
};

export const posterUrl = (path: string | null | undefined) =>
  path ? `https://image.tmdb.org/t/p/w500${path}` : null;
