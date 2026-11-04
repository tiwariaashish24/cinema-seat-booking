package razorpay

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

const (
	baseURL = "https://api.razorpay.com/v1"
)

type RazorpayClient interface {
	CreateOrder(req CreateOrderRequest) (*Order, error)
	VerifyPayment(orderID, paymentID, signature string) (bool, error)
	GetPayment(paymentID string) (*Payment, error)
	IsTestMode() bool
}

type Client struct {
	apiKey     string
	apiSecret  string
	httpClient *http.Client
	testMode   bool
}

func NewClient(apiKey, apiSecret string, testMode bool) *Client {
	return &Client{
		apiKey:    apiKey,
		apiSecret: apiSecret,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
		testMode: testMode,
	}
}

func (c *Client) CreateOrder(req CreateOrderRequest) (*Order, error) {
	body, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	httpReq, err := http.NewRequest("POST", baseURL+"/orders", bytes.NewBuffer(body))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	httpReq.SetBasicAuth(c.apiKey, c.apiSecret)
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("razorpay API error: %s", string(respBody))
	}

	var order Order
	if err := json.Unmarshal(respBody, &order); err != nil {
		return nil, fmt.Errorf("failed to unmarshal response: %w", err)
	}

	return &order, nil
}

func (c *Client) VerifyPayment(orderID, paymentID, signature string) (bool, error) {
	data := orderID + "|" + paymentID
	h := hmac.New(sha256.New, []byte(c.apiSecret))
	h.Write([]byte(data))
	generatedSignature := base64.StdEncoding.EncodeToString(h.Sum(nil))

	return generatedSignature == signature, nil
}

func (c *Client) GetPayment(paymentID string) (*Payment, error) {
	httpReq, err := http.NewRequest("GET", baseURL+"/payments/"+paymentID, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	httpReq.SetBasicAuth(c.apiKey, c.apiSecret)
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("razorpay API error: %s", string(respBody))
	}

	var payment Payment
	if err := json.Unmarshal(respBody, &payment); err != nil {
		return nil, fmt.Errorf("failed to unmarshal response: %w", err)
	}

	return &payment, nil
}

func (c *Client) IsTestMode() bool {
	return c.testMode
}
