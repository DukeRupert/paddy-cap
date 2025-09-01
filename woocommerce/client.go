package woocommerce

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"time"
)

// Client represents a WooCommerce REST API client
type Client struct {
	BaseURL        string
	ConsumerKey    string
	ConsumerSecret string
	HTTPClient     *http.Client
	Version        string // API version, defaults to "v3"
}

// Error represents a WooCommerce API error response
type Error struct {
	Code    string      `json:"code"`
	Message string      `json:"message"`
	Data    interface{} `json:"data"`
}

func (e *Error) Error() string {
	return fmt.Sprintf("WooCommerce API Error %s: %s", e.Code, e.Message)
}

// PaginationInfo holds pagination metadata from response headers
type PaginationInfo struct {
	Total      int
	TotalPages int
	Page       int
	PerPage    int
}

// Response wraps the API response with pagination info
type Response struct {
	Data       interface{}
	Pagination *PaginationInfo
	Headers    http.Header
}

// RequestOptions holds optional parameters for API requests
type RequestOptions struct {
	Page    int
	PerPage int
	Offset  int
	Params  map[string]string
}

// NewClient creates a new WooCommerce client
func NewClient(baseURL, consumerKey, consumerSecret string) *Client {
	return &Client{
		BaseURL:        baseURL,
		ConsumerKey:    consumerKey,
		ConsumerSecret: consumerSecret,
		HTTPClient: &http.Client{
			Timeout: 30 * time.Second,
		},
		Version: "v3",
	}
}

// SetTimeout sets the HTTP client timeout
func (c *Client) SetTimeout(timeout time.Duration) {
	c.HTTPClient.Timeout = timeout
}

// SetVersion sets the API version (default is v3)
func (c *Client) SetVersion(version string) {
	c.Version = version
}

// buildURL constructs the full API endpoint URL
func (c *Client) buildURL(endpoint string, options *RequestOptions) string {
	u, _ := url.Parse(c.BaseURL)
	u.Path = fmt.Sprintf("/wp-json/wc/%s/%s", c.Version, endpoint)

	// Add query parameters
	query := u.Query()

	// Add pagination parameters
	if options != nil {
		if options.Page > 0 {
			query.Set("page", strconv.Itoa(options.Page))
		}
		if options.PerPage > 0 {
			query.Set("per_page", strconv.Itoa(options.PerPage))
		}
		if options.Offset > 0 {
			query.Set("offset", strconv.Itoa(options.Offset))
		}

		// Add custom parameters
		for key, value := range options.Params {
			query.Set(key, value)
		}
	}

	u.RawQuery = query.Encode()
	return u.String()
}

// addAuth adds authentication to the request
func (c *Client) addAuth(req *http.Request) {
	// Using basic auth with consumer key and secret
	req.SetBasicAuth(c.ConsumerKey, c.ConsumerSecret)
}

// makeRequest performs the HTTP request and handles the response
func (c *Client) makeRequest(method, endpoint string, body interface{}, options *RequestOptions) (*Response, error) {
	url := c.buildURL(endpoint, options)

	var reqBody io.Reader
	if body != nil {
		jsonBody, err := json.Marshal(body)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal request body: %w", err)
		}
		reqBody = bytes.NewBuffer(jsonBody)
	}

	req, err := http.NewRequest(method, url, reqBody)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Set headers
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

	// Add authentication
	c.addAuth(req)

	// Make the request
	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	// Read response body
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	// Handle error responses
	if resp.StatusCode >= 400 {
		var wcError Error
		if err := json.Unmarshal(respBody, &wcError); err != nil {
			return nil, fmt.Errorf("HTTP %d: %s", resp.StatusCode, string(respBody))
		}
		return nil, &wcError
	}

	// Parse pagination info from headers
	pagination := &PaginationInfo{}
	if totalStr := resp.Header.Get("X-WP-Total"); totalStr != "" {
		if total, err := strconv.Atoi(totalStr); err == nil {
			pagination.Total = total
		}
	}
	if totalPagesStr := resp.Header.Get("X-WP-TotalPages"); totalPagesStr != "" {
		if totalPages, err := strconv.Atoi(totalPagesStr); err == nil {
			pagination.TotalPages = totalPages
		}
	}

	// Create response wrapper
	response := &Response{
		Pagination: pagination,
		Headers:    resp.Header,
	}

	// Parse JSON response into Data field
	if len(respBody) > 0 {
		var data interface{}
		if err := json.Unmarshal(respBody, &data); err != nil {
			return nil, fmt.Errorf("failed to parse JSON response: %w", err)
		}
		response.Data = data
	}

	return response, nil
}

// GET performs a GET request
func (c *Client) GET(endpoint string, options *RequestOptions) (*Response, error) {
	return c.makeRequest("GET", endpoint, nil, options)
}

// POST performs a POST request
func (c *Client) POST(endpoint string, body interface{}, options *RequestOptions) (*Response, error) {
	return c.makeRequest("POST", endpoint, body, options)
}

// PUT performs a PUT request
func (c *Client) PUT(endpoint string, body interface{}, options *RequestOptions) (*Response, error) {
	return c.makeRequest("PUT", endpoint, body, options)
}

// DELETE performs a DELETE request
func (c *Client) DELETE(endpoint string, options *RequestOptions) (*Response, error) {
	return c.makeRequest("DELETE", endpoint, nil, options)
}

// GetWithPagination is a helper method for paginated GET requests
func (c *Client) GetWithPagination(endpoint string, page, perPage int, params map[string]string) (*Response, error) {
	options := &RequestOptions{
		Page:    page,
		PerPage: perPage,
		Params:  params,
	}
	return c.GET(endpoint, options)
}