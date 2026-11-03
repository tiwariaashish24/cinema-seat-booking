package tmdb

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"os"
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
	apiKey string
	baseURL string
}

func NewClient() *Client {
	return &Client{
		apiKey:  os.Getenv("TMDB_TOKEN"),
		baseURL: "https://api.themoviedb.org/3",
	}
}

func (c *Client) makeRequest(endpoint string) ([]Movie, error) {
	url := fmt.Sprintf("%s%s", c.baseURL, endpoint)

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+c.apiKey)
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("API request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("API returned status %d", resp.StatusCode)
	}

	var movies MovieResponse
	if err := json.NewDecoder(resp.Body).Decode(&movies); err != nil {
		return nil, fmt.Errorf("JSON decoding failed: %w", err)
	}

	return movies.Results, nil
}

func (c *Client) GetUpcomingMovies() ([]Movie, error) {
	return c.makeRequest("/movie/upcoming")
}

func (c *Client) GetPopularMovies() ([]Movie, error) {
	return c.makeRequest("/movie/popular")
}

func (c *Client) GetTrendingMovies() ([]Movie, error) {
	return c.makeRequest("/trending/movie/week")
}

func (c *Client) GetRecommendations(movieID int) ([]Movie, error) {
	return c.makeRequest(fmt.Sprintf("/movie/%d/recommendations", movieID))
}

func (c *Client) SearchMovies(query string) ([]Movie, error) {
	return c.makeRequest(fmt.Sprintf("/search/movie?query=%s", url.QueryEscape(query)))
}
