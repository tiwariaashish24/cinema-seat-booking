package openstreetmap

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"

	"booking/internal/cinema"
)

func TestNearbyUsesRealMapResultsAndCachesCoordinates(t *testing.T) {
	var requests atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests.Add(1)
		if r.Method != http.MethodPost || r.Header.Get("Content-Type") != "application/x-www-form-urlencoded" {
			t.Errorf("unexpected Overpass request: %s %s", r.Method, r.Header.Get("Content-Type"))
		}
		if err := r.ParseForm(); err != nil {
			t.Errorf("parse Overpass form: %v", err)
		}
		if !strings.Contains(r.Form.Get("data"), `["amenity"="cinema"]`) {
			t.Error("query did not restrict results to mapped cinemas")
		}
		fmt.Fprint(w, `{"elements":[{"type":"node","id":2,"lat":28.614,"lon":77.210,"tags":{"name":"Far Cinema","addr:city":"New Delhi"}},{"type":"way","id":1,"center":{"lat":28.61395,"lon":77.209},"tags":{"name":"Near Cinema"}}]}`)
	}))
	defer server.Close()

	client := NewClient()
	client.overpass = server.URL
	origin := &cinema.Coordinates{Latitude: 28.6139, Longitude: 77.2090}
	result, err := client.Nearby(context.Background(), "", origin)
	if err != nil {
		t.Fatalf("Nearby() error = %v", err)
	}
	if result.Source != "OpenStreetMap" || len(result.Cinemas) != 2 {
		t.Fatalf("Nearby() result = %#v, want two real map results", result)
	}
	if result.Cinemas[0].Name != "Near Cinema" || result.Cinemas[0].Distance >= result.Cinemas[1].Distance {
		t.Fatalf("cinemas are not sorted by distance: %#v", result.Cinemas)
	}
	if result.Cinemas[0].MapURL == "" {
		t.Fatal("expected a map URL for the real cinema")
	}

	if _, err := client.Nearby(context.Background(), "", origin); err != nil {
		t.Fatalf("cached Nearby() error = %v", err)
	}
	if got := requests.Load(); got != 1 {
		t.Fatalf("Overpass requests = %d, want cached result after one request", got)
	}
}

func TestNearbyGeocodesWithinIndia(t *testing.T) {
	var requests atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests.Add(1)
		if r.Method == http.MethodGet {
			if r.URL.Query().Get("countrycodes") != "in" || r.URL.Query().Get("q") != "Delhi NCR" {
				t.Errorf("unexpected geocoding query: %v", r.URL.Query())
			}
			if r.Header.Get("User-Agent") == "" {
				t.Error("geocoding request omitted a descriptive User-Agent")
			}
			fmt.Fprint(w, `[{"display_name":"Delhi NCR, India","lat":"28.6139","lon":"77.2090"}]`)
			return
		}
		fmt.Fprint(w, `{"elements":[]}`)
	}))
	defer server.Close()

	client := NewClient()
	client.geocoder = server.URL
	client.overpass = server.URL
	result, err := client.Nearby(context.Background(), "Delhi NCR", nil)
	if err != nil {
		t.Fatalf("Nearby() error = %v", err)
	}
	if result.Location != "Delhi NCR" || len(result.Cinemas) != 0 {
		t.Fatalf("Nearby() result = %#v, want Delhi NCR with no mapped cinemas", result)
	}
	if got := requests.Load(); got != 2 {
		t.Fatalf("geocode and Overpass requests = %d, want 2", got)
	}
}

func TestNearbyRejectsMissingOrInvalidLocation(t *testing.T) {
	client := NewClient()
	for _, test := range []struct {
		name        string
		location    string
		coordinates *cinema.Coordinates
	}{
		{name: "missing location"},
		{name: "out-of-range coordinates", coordinates: &cinema.Coordinates{Latitude: 120, Longitude: 77}},
	} {
		t.Run(test.name, func(t *testing.T) {
			_, err := client.Nearby(context.Background(), test.location, test.coordinates)
			if err == nil {
				t.Fatal("Nearby() expected an input error")
			}
		})
	}
}

func TestNearbyFallsBackAfterTransientGatewayError(t *testing.T) {
	var primaryRequests atomic.Int32
	primary := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		primaryRequests.Add(1)
		http.Error(w, "bad gateway", http.StatusBadGateway)
	}))
	defer primary.Close()

	var fallbackRequests atomic.Int32
	fallback := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		fallbackRequests.Add(1)
		fmt.Fprint(w, `{"elements":[{"type":"node","id":9,"lat":28.614,"lon":77.210,"tags":{"name":"Mapped Cinema"}}]}`)
	}))
	defer fallback.Close()

	client := NewClient()
	client.overpass = primary.URL
	client.fallbacks = []string{fallback.URL}
	result, err := client.Nearby(context.Background(), "", &cinema.Coordinates{Latitude: 28.613, Longitude: 77.209})
	if err != nil {
		t.Fatalf("Nearby() error = %v", err)
	}
	if primaryRequests.Load() != 2 || fallbackRequests.Load() != 1 {
		t.Fatalf("primary requests = %d, fallback requests = %d; want 2 and 1", primaryRequests.Load(), fallbackRequests.Load())
	}
	if len(result.Cinemas) != 1 || result.Cinemas[0].Name != "Mapped Cinema" {
		t.Fatalf("Nearby() result = %#v, want the fallback map result", result)
	}
}
