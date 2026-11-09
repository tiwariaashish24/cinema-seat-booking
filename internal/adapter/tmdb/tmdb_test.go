package tmdb

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
)

func TestGetPopularMoviesUsesRequestedPage(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/movie/popular" {
			t.Errorf("request path = %q, want /movie/popular", r.URL.Path)
		}
		if r.URL.Query().Get("page") != "2" {
			t.Errorf("request page = %q, want 2", r.URL.Query().Get("page"))
		}
		if got := r.Header.Get("Authorization"); got != "Bearer test-token" {
			t.Error("authorization header did not contain the configured bearer token")
		}
		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(MovieResponse{
			Results: []Movie{{ID: 42, Title: "Test Movie"}},
		}); err != nil {
			t.Errorf("encode response: %v", err)
		}
	}))
	defer server.Close()

	client := &Client{apiKey: "test-token", baseURL: server.URL}
	movies, err := client.GetPopularMovies(2)
	if err != nil {
		t.Fatalf("GetPopularMovies(2) error = %v", err)
	}
	if len(movies) != 1 || movies[0].ID != 42 || movies[0].Title != "Test Movie" {
		t.Fatalf("GetPopularMovies(2) = %#v, want test movie", movies)
	}
}

func TestGetPopularMoviesRetriesTransientFailure(t *testing.T) {
	var attempts atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if attempts.Add(1) == 1 {
			http.Error(w, "temporary upstream failure", http.StatusServiceUnavailable)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(MovieResponse{
			Results: []Movie{{ID: 42, Title: "Test Movie"}},
		}); err != nil {
			t.Errorf("encode response: %v", err)
		}
	}))
	defer server.Close()

	client := &Client{apiKey: "test-token", baseURL: server.URL}
	movies, err := client.GetPopularMovies(1)
	if err != nil {
		t.Fatalf("GetPopularMovies(1) error = %v", err)
	}
	if got := attempts.Load(); got != 2 {
		t.Fatalf("request attempts = %d, want 2", got)
	}
	if len(movies) != 1 || movies[0].ID != 42 || movies[0].Title != "Test Movie" {
		t.Fatalf("GetPopularMovies(1) = %#v, want test movie", movies)
	}
}

func TestGetNowPlayingMoviesUsesIndiaRegionAndRequestedPage(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/movie/now_playing" {
			t.Errorf("request path = %q, want /movie/now_playing", r.URL.Path)
		}
		if got := r.URL.Query().Get("region"); got != "IN" {
			t.Errorf("region = %q, want IN", got)
		}
		if got := r.URL.Query().Get("page"); got != "3" {
			t.Errorf("page = %q, want 3", got)
		}
		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(MovieResponse{
			Results: []Movie{{ID: 99, Title: "Now Playing"}},
		}); err != nil {
			t.Errorf("encode response: %v", err)
		}
	}))
	defer server.Close()

	client := &Client{apiKey: "test-token", baseURL: server.URL}
	movies, err := client.GetNowPlayingMovies(3)
	if err != nil {
		t.Fatalf("GetNowPlayingMovies(3) error = %v", err)
	}
	if len(movies) != 1 || movies[0].ID != 99 {
		t.Fatalf("GetNowPlayingMovies(3) = %#v, want the now-playing movie", movies)
	}
}
