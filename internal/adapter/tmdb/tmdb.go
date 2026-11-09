package tmdb

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"
)

type Movie struct {
	ID          int     `json:"id"`
	Title       string  `json:"title"`
	Overview    string  `json:"overview"`
	ReleaseDate string  `json:"release_date"`
	PosterPath  string  `json:"poster_path"`
	Rating      float64 `json:"vote_average"`
}

type MovieResponse struct {
	Results []Movie `json:"results"`
}

type Client struct {
	apiKey  string
	baseURL string
}

const maxAttempts = 3

func NewClient() *Client {
	return &Client{
		apiKey:  os.Getenv("TMDB_TOKEN"),
		baseURL: "https://api.themoviedb.org/3",
	}
}

func (c *Client) makeRequest(endpoint string) ([]Movie, error) {
	requestURL := fmt.Sprintf("%s%s", c.baseURL, endpoint)
	client := &http.Client{Timeout: 10 * time.Second}
	var lastErr error

	for attempt := 0; attempt < maxAttempts; attempt++ {
		req, err := http.NewRequest(http.MethodGet, requestURL, nil)
		if err != nil {
			return nil, fmt.Errorf("create request: %w", err)
		}
		req.Header.Set("Authorization", "Bearer "+c.apiKey)
		req.Header.Set("Accept", "application/json")

		resp, err := client.Do(req)
		if err != nil {
			lastErr = fmt.Errorf("API request failed: %w", err)
		} else if resp.StatusCode != http.StatusOK {
			body, readErr := io.ReadAll(io.LimitReader(resp.Body, 4096))
			resp.Body.Close()
			if readErr != nil {
				lastErr = fmt.Errorf("read API error response: %w", readErr)
			} else {
				lastErr = fmt.Errorf("API returned status %d: %s", resp.StatusCode, strings.TrimSpace(string(body)))
			}
			if resp.StatusCode != http.StatusTooManyRequests && resp.StatusCode < http.StatusInternalServerError {
				return nil, lastErr
			}
		} else {
			var movies MovieResponse
			decodeErr := json.NewDecoder(resp.Body).Decode(&movies)
			resp.Body.Close()
			if decodeErr != nil {
				return nil, fmt.Errorf("JSON decoding failed: %w", decodeErr)
			}
			return movies.Results, nil
		}

		if attempt < maxAttempts-1 {
			time.Sleep(time.Duration(attempt+1) * 150 * time.Millisecond)
		}
	}

	return nil, lastErr
}

func (c *Client) makePagedRequest(endpoint string, page int) ([]Movie, error) {
	requestURL, err := url.Parse(endpoint)
	if err != nil {
		return nil, fmt.Errorf("parse request URL: %w", err)
	}

	query := requestURL.Query()
	query.Set("page", fmt.Sprintf("%d", page))
	requestURL.RawQuery = query.Encode()
	return c.makeRequest(requestURL.String())
}

func (c *Client) GetUpcomingMovies(page int) ([]Movie, error) {
	return c.makePagedRequest("/movie/upcoming", page)
}

func (c *Client) GetPopularMovies(page int) ([]Movie, error) {
	return c.makePagedRequest("/movie/popular", page)
}

func (c *Client) GetTrendingMovies(page int) ([]Movie, error) {
	return c.makePagedRequest("/trending/movie/week", page)
}

func (c *Client) GetRecommendations(movieID int) ([]Movie, error) {
	return c.makeRequest(fmt.Sprintf("/movie/%d/recommendations", movieID))
}

func (c *Client) SearchMovies(query string) ([]Movie, error) {
	return c.makeRequest(fmt.Sprintf("/search/movie?query=%s", url.QueryEscape(query)))
}
