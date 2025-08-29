// Package orderspace provides a Go client for the Orderspace REST API
package orderspace

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

// Client represents an Orderspace REST API client
type Client struct {
	BaseURL      string
	ClientID     string
	ClientSecret string
	HTTPClient   *http.Client
	accessToken  string
	tokenExpiry  time.Time
}

// Error represents an Orderspace API error response
type Error struct {
	Message string `json:"message"`
	Code    int    `json:"code,omitempty"`
}

func (e *Error) Error() string {
	if e.Code != 0 {
		return fmt.Sprintf("Orderspace API Error %d: %s", e.Code, e.Message)
	}
	return fmt.Sprintf("Orderspace API Error: %s", e.Message)
}

// TokenResponse represents the OAuth token response
type TokenResponse struct {
	AccessToken string `json:"access_token"`
	TokenType   string `json:"token_type"`
	ExpiresIn   int    `json:"expires_in"`
	Scope       string `json:"scope"`
}

// PaginationInfo holds pagination metadata
type PaginationInfo struct {
	Limit         int
	StartingAfter string
	HasMore       bool
}

// Response wraps the API response with pagination info
type Response struct {
	Data       interface{}
	Pagination *PaginationInfo
	Headers    http.Header
}

// RequestOptions holds optional parameters for API requests
type RequestOptions struct {
	Limit         int
	StartingAfter string
	Params        map[string]string
}

// NewClient creates a new Orderspace client
func NewClient(baseUrl, clientID, clientSecret string) *Client {
	return &Client{
		BaseURL:      baseUrl,
		ClientID:     clientID,
		ClientSecret: clientSecret,
		HTTPClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// SetTimeout sets the HTTP client timeout
func (c *Client) SetTimeout(timeout time.Duration) {
	c.HTTPClient.Timeout = timeout
}

// getAccessToken obtains a new access token using OAuth2 client credentials flow
func (c *Client) getAccessToken() error {
	tokenURL := "https://identity.orderspace.com/oauth/token"
	
	data := url.Values{}
	data.Set("client_id", c.ClientID)
	data.Set("client_secret", c.ClientSecret)
	data.Set("grant_type", "client_credentials")
	
	req, err := http.NewRequest("POST", tokenURL, bytes.NewBufferString(data.Encode()))
	if err != nil {
		return fmt.Errorf("failed to create token request: %w", err)
	}
	
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	
	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to get access token: %w", err)
	}
	defer resp.Body.Close()
	
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read token response: %w", err)
	}
	
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("token request failed with status %d: %s", resp.StatusCode, string(body))
	}
	
	var tokenResp TokenResponse
	if err := json.Unmarshal(body, &tokenResp); err != nil {
		return fmt.Errorf("failed to parse token response: %w", err)
	}
	
	c.accessToken = tokenResp.AccessToken
	// Set expiry to be 30 seconds before actual expiry to allow for refresh
	c.tokenExpiry = time.Now().Add(time.Duration(tokenResp.ExpiresIn-30) * time.Second)
	
	return nil
}

// ensureValidToken ensures we have a valid access token
func (c *Client) ensureValidToken() error {
	if c.accessToken == "" || time.Now().After(c.tokenExpiry) {
		return c.getAccessToken()
	}
	return nil
}

// buildURL constructs the full API endpoint URL
func (c *Client) buildURL(endpoint string, options *RequestOptions) string {
	u, _ := url.Parse(c.BaseURL)
	u.Path = u.Path + "/" + endpoint
	
	// Add query parameters
	query := u.Query()
	
	if options != nil {
		if options.Limit > 0 {
			query.Set("limit", strconv.Itoa(options.Limit))
		}
		if options.StartingAfter != "" {
			query.Set("starting_after", options.StartingAfter)
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
func (c *Client) addAuth(req *http.Request) error {
	if err := c.ensureValidToken(); err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+c.accessToken)
	return nil
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
	if err := c.addAuth(req); err != nil {
		return nil, fmt.Errorf("failed to add authentication: %w", err)
	}
	
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
		var apiError Error
		if err := json.Unmarshal(respBody, &apiError); err != nil {
			return nil, fmt.Errorf("HTTP %d: %s", resp.StatusCode, string(respBody))
		}
		apiError.Code = resp.StatusCode
		return nil, &apiError
	}
	
	// Create response wrapper
	response := &Response{
		Headers: resp.Header,
		Pagination: &PaginationInfo{
			// Orderspace uses cursor-based pagination
			// We'll need to determine HasMore from the response data
		},
	}
	
	// Parse JSON response into Data field
	if len(respBody) > 0 {
		var data interface{}
		if err := json.Unmarshal(respBody, &data); err != nil {
			return nil, fmt.Errorf("failed to parse JSON response: %w", err)
		}
		response.Data = data
		
		// For pagination, check if we got a full page (indicating there might be more)
		if options != nil && options.Limit > 0 {
			if dataSlice, ok := data.([]interface{}); ok {
				response.Pagination.HasMore = len(dataSlice) == options.Limit
				response.Pagination.Limit = options.Limit
				response.Pagination.StartingAfter = options.StartingAfter
			}
		}
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

// PATCH performs a PATCH request
func (c *Client) PATCH(endpoint string, body interface{}, options *RequestOptions) (*Response, error) {
	return c.makeRequest("PATCH", endpoint, body, options)
}

// GetWithPagination is a helper method for paginated GET requests
func (c *Client) GetWithPagination(endpoint string, limit int, startingAfter string, params map[string]string) (*Response, error) {
	options := &RequestOptions{
		Limit:         limit,
		StartingAfter: startingAfter,
		Params:        params,
	}
	return c.GET(endpoint, options)
}

// GetNextPage gets the next page of results using cursor pagination
func (c *Client) GetNextPage(endpoint string, lastID string, limit int, params map[string]string) (*Response, error) {
	return c.GetWithPagination(endpoint, limit, lastID, params)
}