package main

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestParseMoviePage(t *testing.T) {
	tests := []struct {
		name      string
		query     string
		wantPage  int
		wantError bool
	}{
		{name: "default page", wantPage: 1},
		{name: "requested page", query: "?page=25", wantPage: 25},
		{name: "zero page", query: "?page=0", wantError: true},
		{name: "page above TMDB limit", query: "?page=501", wantError: true},
		{name: "invalid page", query: "?page=first", wantError: true},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			request := httptest.NewRequest(http.MethodGet, "/api/movies/popular"+test.query, nil)
			page, err := parseMoviePage(request)
			if (err != nil) != test.wantError {
				t.Fatalf("parseMoviePage() error = %v, wantError %v", err, test.wantError)
			}
			if err == nil && page != test.wantPage {
				t.Fatalf("parseMoviePage() = %d, want %d", page, test.wantPage)
			}
		})
	}
}
