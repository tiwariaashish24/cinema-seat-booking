package openstreetmap

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"net/http"
	"net/url"
	"sort"
	"strings"
	"sync"
	"time"

	"booking/internal/cinema"
)

const (
	nominatimURL      = "https://nominatim.openstreetmap.org/search"
	overpassURL       = "https://overpass-api.de/api/interpreter"
	cacheTTL          = time.Hour
	maxPrimaryRetries = 1
)

type Client struct {
	httpClient *http.Client
	geocoder   string
	overpass   string
	fallbacks  []string

	mu            sync.Mutex
	geocodeMu     sync.Mutex
	lastGeocodeAt time.Time
	cache         map[string]cachedResult
}

type cachedResult struct {
	result    cinema.NearbyCinemaResult
	expiresAt time.Time
}

type geocodeResult struct {
	DisplayName string `json:"display_name"`
	Latitude    string `json:"lat"`
	Longitude   string `json:"lon"`
}

type overpassResponse struct {
	Elements []overpassElement `json:"elements"`
}

type overpassElement struct {
	Type   string            `json:"type"`
	ID     int64             `json:"id"`
	Lat    float64           `json:"lat"`
	Lon    float64           `json:"lon"`
	Center *overpassCenter   `json:"center"`
	Tags   map[string]string `json:"tags"`
}

type overpassCenter struct {
	Lat float64 `json:"lat"`
	Lon float64 `json:"lon"`
}

func NewClient() *Client {
	return &Client{
		httpClient: &http.Client{Timeout: 5 * time.Second},
		geocoder:   nominatimURL,
		overpass:   overpassURL,
		fallbacks: []string{
			"https://overpass.kumi.systems/api/interpreter",
			"https://overpass.private.coffee/api/interpreter",
		},
		cache: make(map[string]cachedResult),
	}
}

func (c *Client) Nearby(ctx context.Context, location string, coordinates *cinema.Coordinates) (cinema.NearbyCinemaResult, error) {
	location = strings.TrimSpace(location)
	if len(location) > 160 {
		return cinema.NearbyCinemaResult{}, fmt.Errorf("location must be at most 160 characters")
	}
	if coordinates == nil && location == "" {
		return cinema.NearbyCinemaResult{}, fmt.Errorf("provide a location or coordinates")
	}

	cacheKey := strings.ToLower(location)
	if coordinates != nil {
		if !validCoordinates(*coordinates) {
			return cinema.NearbyCinemaResult{}, fmt.Errorf("coordinates are out of range")
		}
		cacheKey = fmt.Sprintf("%.3f,%.3f", coordinates.Latitude, coordinates.Longitude)
	}
	if result, ok := c.cached(cacheKey); ok {
		return result, nil
	}

	origin := cinema.Coordinates{}
	resolvedLocation := location
	if coordinates != nil {
		origin = *coordinates
		resolvedLocation = "Your location"
	} else {
		geocoded, err := c.geocode(ctx, location)
		if err != nil {
			return cinema.NearbyCinemaResult{}, err
		}
		origin = cinema.Coordinates{Latitude: geocoded.Latitude, Longitude: geocoded.Longitude}
	}

	cinemas, err := c.findCinemas(ctx, origin)
	if err != nil {
		return cinema.NearbyCinemaResult{}, err
	}
	result := cinema.NearbyCinemaResult{
		Location:    resolvedLocation,
		Source:      "OpenStreetMap",
		Attribution: "Map data © OpenStreetMap contributors",
		Cinemas:     cinemas,
	}
	c.storeCache(cacheKey, result)
	return result, nil
}

func (c *Client) cached(key string) (cinema.NearbyCinemaResult, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	entry, ok := c.cache[key]
	if !ok || time.Now().After(entry.expiresAt) {
		delete(c.cache, key)
		return cinema.NearbyCinemaResult{}, false
	}
	return entry.result, true
}

func (c *Client) storeCache(key string, result cinema.NearbyCinemaResult) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if len(c.cache) >= 256 {
		now := time.Now()
		for cachedKey, entry := range c.cache {
			if now.After(entry.expiresAt) {
				delete(c.cache, cachedKey)
			}
		}
		if len(c.cache) >= 256 {
			for cachedKey := range c.cache {
				delete(c.cache, cachedKey)
				break
			}
		}
	}
	c.cache[key] = cachedResult{result: result, expiresAt: time.Now().Add(cacheTTL)}
}

func (c *Client) geocode(ctx context.Context, location string) (cinema.Coordinates, error) {
	query := url.Values{
		"format":       {"jsonv2"},
		"limit":        {"1"},
		"countrycodes": {"in"},
		"q":            {location},
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.geocoder+"?"+query.Encode(), nil)
	if err != nil {
		return cinema.Coordinates{}, fmt.Errorf("create location lookup request: %w", err)
	}
	req.Header.Set("User-Agent", "CinemaSeatBooking/1.0 (OpenStreetMap Nominatim client)")
	req.Header.Set("Accept", "application/json")

	c.geocodeMu.Lock()
	wait := time.Second - time.Since(c.lastGeocodeAt)
	if wait > 0 {
		timer := time.NewTimer(wait)
		select {
		case <-ctx.Done():
			timer.Stop()
			c.geocodeMu.Unlock()
			return cinema.Coordinates{}, ctx.Err()
		case <-timer.C:
		}
	}
	c.lastGeocodeAt = time.Now()
	c.geocodeMu.Unlock()

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return cinema.Coordinates{}, fmt.Errorf("location lookup failed: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return cinema.Coordinates{}, fmt.Errorf("location lookup returned status %d", resp.StatusCode)
	}
	var results []geocodeResult
	if err := json.NewDecoder(resp.Body).Decode(&results); err != nil {
		return cinema.Coordinates{}, fmt.Errorf("decode location lookup: %w", err)
	}
	if len(results) == 0 {
		return cinema.Coordinates{}, fmt.Errorf("could not find that location in India")
	}

	latitude, err := parseCoordinate(results[0].Latitude, -90, 90)
	if err != nil {
		return cinema.Coordinates{}, fmt.Errorf("invalid latitude returned by location lookup: %w", err)
	}
	longitude, err := parseCoordinate(results[0].Longitude, -180, 180)
	if err != nil {
		return cinema.Coordinates{}, fmt.Errorf("invalid longitude returned by location lookup: %w", err)
	}
	return cinema.Coordinates{Latitude: latitude, Longitude: longitude}, nil
}

func (c *Client) findCinemas(ctx context.Context, origin cinema.Coordinates) ([]cinema.NearbyCinema, error) {
	query := fmt.Sprintf(`[out:json][timeout:10];(node["amenity"="cinema"](around:25000,%.6f,%.6f);way["amenity"="cinema"](around:25000,%.6f,%.6f);relation["amenity"="cinema"](around:25000,%.6f,%.6f););out center tags;`,
		origin.Latitude, origin.Longitude,
		origin.Latitude, origin.Longitude,
		origin.Latitude, origin.Longitude,
	)
	endpoints := []string{c.overpass}
	if len(c.fallbacks) > 0 {
		endpoints = append(endpoints, c.fallbacks...)
	}
	var lastErr error
	var data overpassResponse
	for endpointIndex, endpoint := range endpoints {
		retries := 0
		if endpointIndex == 0 {
			retries = maxPrimaryRetries
		}
		for retry := 0; retry <= retries; retry++ {
			data, lastErr = c.requestCinemas(ctx, endpoint, query)
			if lastErr == nil {
				return c.mapCinemas(data, origin), nil
			}
			if ctx.Err() != nil {
				return nil, fmt.Errorf("nearby cinema search timed out; please try again or search another area")
			}
			if retry < retries && retryableEndpointError(lastErr) {
				timer := time.NewTimer(250 * time.Millisecond)
				select {
				case <-ctx.Done():
					timer.Stop()
					return nil, fmt.Errorf("nearby cinema search timed out; please try again or search another area")
				case <-timer.C:
				}
				continue
			}
		}
		if ctx.Err() != nil {
			return nil, fmt.Errorf("nearby cinema search timed out; please try again or search another area")
		}
	}
	return nil, fmt.Errorf("nearby cinema data is temporarily unavailable; please try again in a few moments (%w)", lastErr)
}

func (c *Client) requestCinemas(ctx context.Context, endpoint, query string) (overpassResponse, error) {
	form := url.Values{"data": {query}}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, strings.NewReader(form.Encode()))
	if err != nil {
		return overpassResponse{}, fmt.Errorf("create cinema lookup request: %w", err)
	}
	req.Header.Set("User-Agent", "CinemaSeatBooking/1.0 (OpenStreetMap cinema lookup)")
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Accept", "application/json")
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return overpassResponse{}, fmt.Errorf("nearby cinema lookup request failed: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return overpassResponse{}, fmt.Errorf("nearby cinema lookup returned status %d", resp.StatusCode)
	}
	var data overpassResponse
	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		return overpassResponse{}, fmt.Errorf("decode nearby cinema results: %w", err)
	}
	return data, nil
}

func retryableEndpointError(err error) bool {
	if err == nil {
		return false
	}
	message := err.Error()
	return strings.Contains(message, "request failed") ||
		strings.Contains(message, "decode nearby cinema results") ||
		strings.Contains(message, "returned status 429") ||
		strings.Contains(message, "returned status 502") ||
		strings.Contains(message, "returned status 503") ||
		strings.Contains(message, "returned status 504")
}

func (c *Client) mapCinemas(data overpassResponse, origin cinema.Coordinates) []cinema.NearbyCinema {
	cinemas := make([]cinema.NearbyCinema, 0, len(data.Elements))
	for _, element := range data.Elements {
		name := strings.TrimSpace(element.Tags["name"])
		if name == "" {
			continue
		}
		latitude, longitude := element.Lat, element.Lon
		if element.Center != nil {
			latitude, longitude = element.Center.Lat, element.Center.Lon
		}
		if latitude == 0 && longitude == 0 {
			continue
		}
		address := strings.TrimSpace(element.Tags["addr:full"])
		if address == "" {
			address = strings.Join(nonEmpty(
				element.Tags["addr:housenumber"],
				element.Tags["addr:street"],
				element.Tags["addr:suburb"],
				element.Tags["addr:city"],
			), ", ")
		}
		cinemas = append(cinemas, cinema.NearbyCinema{
			ID:        fmt.Sprintf("%s/%d", element.Type, element.ID),
			Name:      name,
			Address:   address,
			Latitude:  latitude,
			Longitude: longitude,
			Distance:  haversineKm(origin.Latitude, origin.Longitude, latitude, longitude),
			MapURL:    fmt.Sprintf("https://www.openstreetmap.org/?mlat=%.6f&mlon=%.6f#map=17/%.6f/%.6f", latitude, longitude, latitude, longitude),
		})
	}
	sort.Slice(cinemas, func(i, j int) bool {
		return cinemas[i].Distance < cinemas[j].Distance
	})
	if len(cinemas) > 20 {
		cinemas = cinemas[:20]
	}
	return cinemas
}

func validCoordinates(coordinates cinema.Coordinates) bool {
	return coordinates.Latitude >= -90 && coordinates.Latitude <= 90 &&
		coordinates.Longitude >= -180 && coordinates.Longitude <= 180
}

func parseCoordinate(value string, min, max float64) (float64, error) {
	var coordinate float64
	if _, err := fmt.Sscanf(value, "%f", &coordinate); err != nil || math.IsNaN(coordinate) || coordinate < min || coordinate > max {
		return 0, fmt.Errorf("coordinate is out of range")
	}
	return coordinate, nil
}

func haversineKm(lat1, lon1, lat2, lon2 float64) float64 {
	const earthRadiusKm = 6371
	toRadians := math.Pi / 180
	dLat := (lat2 - lat1) * toRadians
	dLon := (lon2 - lon1) * toRadians
	a := math.Sin(dLat/2)*math.Sin(dLat/2) +
		math.Cos(lat1*toRadians)*math.Cos(lat2*toRadians)*math.Sin(dLon/2)*math.Sin(dLon/2)
	return earthRadiusKm * 2 * math.Atan2(math.Sqrt(a), math.Sqrt(1-a))
}

func nonEmpty(values ...string) []string {
	result := make([]string, 0, len(values))
	for _, value := range values {
		if value = strings.TrimSpace(value); value != "" {
			result = append(result, value)
		}
	}
	return result
}
