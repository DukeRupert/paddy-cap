package orderspace

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
)

// Order represents an Orderspace order
type Order struct {
	ID               string              `json:"id"`
	Number           int                 `json:"number"`
	Created          string              `json:"created"`
	Status           string              `json:"status"`
	CustomerID       string              `json:"customer_id"`
	CompanyName      string              `json:"company_name"`
	Phone            string              `json:"phone"`
	EmailAddresses   OrderEmailAddresses `json:"email_addresses"`
	CreatedBy        string              `json:"created_by"`
	DeliveryDate     string              `json:"delivery_date"`
	Reference        string              `json:"reference"`
	InternalNote     string              `json:"internal_note"`
	CustomerPONumber string              `json:"customer_po_number"`
	CustomerNote     string              `json:"customer_note"`
	StandingOrderID  string              `json:"standing_order_id"`
	ShippingType     string              `json:"shipping_type"`
	ShippingAddress  OrderAddress        `json:"shipping_address"`
	BillingAddress   OrderAddress        `json:"billing_address"`
	OrderLines       []OrderLine         `json:"order_lines"`
	Currency         string              `json:"currency"`
	NetTotal         float64             `json:"net_total"`
	GrossTotal       float64             `json:"gross_total"`
}

// OrderEmailAddresses represents the email addresses for different purposes
type OrderEmailAddresses struct {
	Orders     string `json:"orders"`
	Dispatches string `json:"dispatches"`
	Invoices   string `json:"invoices"`
}

// OrderAddress represents billing or shipping address
type OrderAddress struct {
	CompanyName string `json:"company_name"`
	ContactName string `json:"contact_name"`
	Line1       string `json:"line1"`
	Line2       string `json:"line2"`
	City        string `json:"city"`
	State       string `json:"state"`
	PostalCode  string `json:"postal_code"`
	Country     string `json:"country"`
}

// OrderLine represents a line item in an order
type OrderLine struct {
	ID               string                    `json:"id"`
	SKU              string                    `json:"sku"`
	Name             string                    `json:"name"`
	Options          string                    `json:"options"`
	GroupingCategory OrderLineGroupingCategory `json:"grouping_category"`
	Shipping         bool                      `json:"shipping"`
	Quantity         int                       `json:"quantity"`
	UnitPrice        float64                   `json:"unit_price"`
	SubTotal         float64                   `json:"sub_total"`
	TaxRateID        string                    `json:"tax_rate_id"`
	TaxName          string                    `json:"tax_name"`
	TaxRate          float64                   `json:"tax_rate"`
	TaxAmount        float64                   `json:"tax_amount"`
	PreorderWindowID string                    `json:"preorder_window_id"`
	OnHold           bool                      `json:"on_hold"`
	Invoiced         int                       `json:"invoiced"`
	Paid             int                       `json:"paid"`
	Dispatched       int                       `json:"dispatched"`
}

// OrderLineGroupingCategory represents the category grouping for an order line
type OrderLineGroupingCategory struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

// OrdersResponse represents the response when fetching multiple orders
type OrdersResponse struct {
	Orders     []Order
	Pagination *PaginationInfo
	Headers    http.Header
}

// OrderListOptions holds filtering options for listing orders
type OrderListOptions struct {
	// Pagination
	Limit         int
	StartingAfter string

	// Filtering
	Status       string // Order status filter
	CustomerID   string // Filter by customer ID
	CreatedSince string // Filter orders created since this date (ISO 8601)
	CreatedUntil string // Filter orders created until this date (ISO 8601)
	UpdatedSince string // Filter orders updated since this date (ISO 8601)
	UpdatedUntil string // Filter orders updated until this date (ISO 8601)

	// Additional custom parameters
	Params map[string]string
}

// ListOrders retrieves orders with optional filtering
func (c *Client) ListOrders(options *OrderListOptions) (*OrdersResponse, error) {
	params := make(map[string]string)
	requestOptions := &RequestOptions{
		Params: params,
	}

	if options != nil {
		// Set pagination
		requestOptions.Limit = options.Limit
		requestOptions.StartingAfter = options.StartingAfter

		// Set filtering parameters
		if options.Status != "" {
			params["status"] = options.Status
		}
		if options.CustomerID != "" {
			params["customer_id"] = options.CustomerID
		}
		if options.CreatedSince != "" {
			params["created_since"] = options.CreatedSince
		}
		if options.CreatedUntil != "" {
			params["created_until"] = options.CreatedUntil
		}
		if options.UpdatedSince != "" {
			params["updated_since"] = options.UpdatedSince
		}
		if options.UpdatedUntil != "" {
			params["updated_until"] = options.UpdatedUntil
		}

		// Add any additional custom parameters
		for key, value := range options.Params {
			params[key] = value
		}
	}

	response, err := c.GET("orders", requestOptions)
	if err != nil {
		return nil, err
	}

	// Parse the response data into Order structs
	// Note: The API response might be wrapped in an "orders" field or be a direct array
	var orders []Order
	if response.Data != nil {
		jsonData, err := json.Marshal(response.Data)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal response data: %w", err)
		}

		// Try to unmarshal as direct array first
		if err := json.Unmarshal(jsonData, &orders); err != nil {
			// If that fails, try to unmarshal as wrapped response
			var wrappedResponse struct {
				Orders []Order `json:"orders"`
			}
			if err2 := json.Unmarshal(jsonData, &wrappedResponse); err2 != nil {
				return nil, fmt.Errorf("failed to unmarshal orders: %w", err)
			}
			orders = wrappedResponse.Orders
		}
	}

	return &OrdersResponse{
		Orders:     orders,
		Pagination: response.Pagination,
		Headers:    response.Headers,
	}, nil
}

// GetOrder retrieves a single order by ID
// GetOrder retrieves a single order by ID
func (c *Client) GetOrder(orderID string) (*Order, error) {
	slog.Info("GetOrder called", "orderID", orderID)

	endpoint := fmt.Sprintf("orders/%s", orderID)
	slog.Debug("Making GET request", "endpoint", endpoint)

	response, err := c.GET(endpoint, nil)
	if err != nil {
		slog.Error("GET request failed", "endpoint", endpoint, "error", err)
		return nil, err
	}
	slog.Debug("GET request successful", "response_status", "ok")

	if response.Data == nil {
		slog.Error("Response data is nil", "endpoint", endpoint)
		return nil, fmt.Errorf("no data in response")
	}
	slog.Debug("Response data present", "data_type", fmt.Sprintf("%T", response.Data))

	jsonData, err := json.Marshal(response.Data)
	if err != nil {
		slog.Error("Failed to marshal response data", "error", err, "data_type", fmt.Sprintf("%T", response.Data))
		return nil, fmt.Errorf("failed to marshal response data: %w", err)
	}
	slog.Debug("Successfully marshaled response data", "json_length", len(jsonData), "json_preview", string(jsonData[:min(200, len(jsonData))]))

	// Orderspace API returns single orders wrapped in an "order" object
	var wrappedResponse struct {
		Order Order `json:"order"`
	}
	if err := json.Unmarshal(jsonData, &wrappedResponse); err != nil {
		slog.Error("Failed to unmarshal order", "error", err, "json_data", string(jsonData))
		return nil, fmt.Errorf("failed to unmarshal order: %w", err)
	}
	slog.Debug("Successfully unmarshaled order", "order_id", wrappedResponse.Order.ID)

	slog.Info("GetOrder completed successfully", "orderID", orderID, "retrieved_order_id", wrappedResponse.Order.ID)
	return &wrappedResponse.Order, nil
}

// Helper methods for common order queries

// GetAllOrders retrieves orders with basic pagination
func (c *Client) GetAllOrders(limit int, startingAfter string) (*OrdersResponse, error) {
	options := &OrderListOptions{
		Limit:         limit,
		StartingAfter: startingAfter,
	}
	return c.ListOrders(options)
}

// GetOrdersByStatus retrieves orders filtered by status
func (c *Client) GetOrdersByStatus(status string, limit int, startingAfter string) (*OrdersResponse, error) {
	options := &OrderListOptions{
		Status:        status,
		Limit:         limit,
		StartingAfter: startingAfter,
	}
	return c.ListOrders(options)
}

// GetOrdersByCustomer retrieves orders for a specific customer
func (c *Client) GetOrdersByCustomer(customerID string, limit int, startingAfter string) (*OrdersResponse, error) {
	options := &OrderListOptions{
		CustomerID:    customerID,
		Limit:         limit,
		StartingAfter: startingAfter,
	}
	return c.ListOrders(options)
}

// GetRecentOrders retrieves orders created since a specific date
func (c *Client) GetRecentOrders(createdSince string, limit int, startingAfter string) (*OrdersResponse, error) {
	options := &OrderListOptions{
		CreatedSince:  createdSince,
		Limit:         limit,
		StartingAfter: startingAfter,
	}
	return c.ListOrders(options)
}

// GetOrdersInDateRange retrieves orders within a date range
func (c *Client) GetOrdersInDateRange(createdSince, createdUntil string, limit int, startingAfter string) (*OrdersResponse, error) {
	options := &OrderListOptions{
		CreatedSince:  createdSince,
		CreatedUntil:  createdUntil,
		Limit:         limit,
		StartingAfter: startingAfter,
	}
	return c.ListOrders(options)
}

// GetLast10Orders is a convenience method to get the last 10 orders
func (c *Client) GetLast10Orders() (*OrdersResponse, error) {
	return c.GetAllOrders(10, "")
}