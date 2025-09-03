package woocommerce

import (
	"encoding/json"
	"fmt"
	"net/http"
)

// Order represents a WooCommerce order
type Order struct {
	ID                   int                    `json:"id"`
	ParentID             int                    `json:"parent_id"`
	Number               string                 `json:"number"`
	OrderKey             string                 `json:"order_key"`
	CreatedVia           string                 `json:"created_via"`
	Version              string                 `json:"version"`
	Status               string                 `json:"status"`
	Currency             string                 `json:"currency"`
	DateCreated          string                 `json:"date_created"`
	DateCreatedGMT       string                 `json:"date_created_gmt"`
	DateModified         string                 `json:"date_modified"`
	DateModifiedGMT      string                 `json:"date_modified_gmt"`
	DiscountTotal        string                 `json:"discount_total"`
	DiscountTax          string                 `json:"discount_tax"`
	ShippingTotal        string                 `json:"shipping_total"`
	ShippingTax          string                 `json:"shipping_tax"`
	CartTax              string                 `json:"cart_tax"`
	Total                string                 `json:"total"`
	TotalTax             string                 `json:"total_tax"`
	PricesIncludeTax     bool                   `json:"prices_include_tax"`
	CustomerID           int                    `json:"customer_id"`
	CustomerIPAddress    string                 `json:"customer_ip_address"`
	CustomerUserAgent    string                 `json:"customer_user_agent"`
	CustomerNote         string                 `json:"customer_note"`
	Billing              OrderAddress           `json:"billing"`
	Shipping             OrderAddress           `json:"shipping"`
	PaymentMethod        string                 `json:"payment_method"`
	PaymentMethodTitle   string                 `json:"payment_method_title"`
	TransactionID        string                 `json:"transaction_id"`
	DatePaid             *string                `json:"date_paid"`
	DatePaidGMT          *string                `json:"date_paid_gmt"`
	DateCompleted        *string                `json:"date_completed"`
	DateCompletedGMT     *string                `json:"date_completed_gmt"`
	CartHash             string                 `json:"cart_hash"`
	MetaData             []OrderMetaData        `json:"meta_data"`
	LineItems            []OrderLineItem        `json:"line_items"`
	TaxLines             []OrderTaxLine         `json:"tax_lines"`
	ShippingLines        []OrderShippingLine    `json:"shipping_lines"`
	FeeLines             []OrderFeeLine         `json:"fee_lines"`
	CouponLines          []OrderCouponLine      `json:"coupon_lines"`
	Refunds              []OrderRefund          `json:"refunds"`
	Links                map[string]interface{} `json:"_links"`
}

// OrderAddress represents billing or shipping address
type OrderAddress struct {
	FirstName string `json:"first_name"`
	LastName  string `json:"last_name"`
	Company   string `json:"company"`
	Address1  string `json:"address_1"`
	Address2  string `json:"address_2"`
	City      string `json:"city"`
	State     string `json:"state"`
	Postcode  string `json:"postcode"`
	Country   string `json:"country"`
	Email     string `json:"email,omitempty"` // Only in billing address
	Phone     string `json:"phone,omitempty"` // Only in billing address
}

// OrderMetaData represents order meta data
type OrderMetaData struct {
	ID    int         `json:"id"`
	Key   string      `json:"key"`
	Value interface{} `json:"value"`
}

// OrderLineItem represents a line item in an order
type OrderLineItem struct {
	ID          int                   `json:"id"`
	Name        string                `json:"name"`
	ProductID   int                   `json:"product_id"`
	VariationID int                   `json:"variation_id"`
	Quantity    int                   `json:"quantity"`
	TaxClass    string                `json:"tax_class"`
	Subtotal    string                `json:"subtotal"`
	SubtotalTax string                `json:"subtotal_tax"`
	Total       string                `json:"total"`
	TotalTax    string                `json:"total_tax"`
	Taxes       []OrderLineItemTax    `json:"taxes"`
	MetaData    []OrderMetaData       `json:"meta_data"`
	SKU         string                `json:"sku"`
	Price       interface{}           `json:"price"` // Can be int or float
}

// OrderLineItemTax represents tax information for a line item
type OrderLineItemTax struct {
	ID       int    `json:"id"`
	Total    string `json:"total"`
	Subtotal string `json:"subtotal"`
}

// OrderTaxLine represents tax line information
type OrderTaxLine struct {
	ID               int             `json:"id"`
	RateCode         string          `json:"rate_code"`
	RateID           int             `json:"rate_id"`
	Label            string          `json:"label"`
	Compound         bool            `json:"compound"`
	TaxTotal         string          `json:"tax_total"`
	ShippingTaxTotal string          `json:"shipping_tax_total"`
	MetaData         []OrderMetaData `json:"meta_data"`
}

// OrderShippingLine represents shipping line information
type OrderShippingLine struct {
	ID          int             `json:"id"`
	MethodTitle string          `json:"method_title"`
	MethodID    string          `json:"method_id"`
	Total       string          `json:"total"`
	TotalTax    string          `json:"total_tax"`
	Taxes       []interface{}   `json:"taxes"`
	MetaData    []OrderMetaData `json:"meta_data"`
}

// OrderFeeLine represents fee line information
type OrderFeeLine struct {
	ID        int             `json:"id"`
	Name      string          `json:"name"`
	TaxClass  string          `json:"tax_class"`
	TaxStatus string          `json:"tax_status"`
	Total     string          `json:"total"`
	TotalTax  string          `json:"total_tax"`
	Taxes     []interface{}   `json:"taxes"`
	MetaData  []OrderMetaData `json:"meta_data"`
}

// OrderCouponLine represents coupon line information
type OrderCouponLine struct {
	ID          int             `json:"id"`
	Code        string          `json:"code"`
	Discount    string          `json:"discount"`
	DiscountTax string          `json:"discount_tax"`
	MetaData    []OrderMetaData `json:"meta_data"`
}

// OrderRefund represents refund information
type OrderRefund struct {
	ID     int    `json:"id"`
	Refund string `json:"refund"`
	Total  string `json:"total"`
}

// OrdersResponse represents the response when fetching multiple orders
type OrdersResponse struct {
	Orders     []Order
	Pagination *PaginationInfo
	Headers    http.Header
}

// OrderListOptions holds all possible filtering options for listing orders
type OrderListOptions struct {
	// Pagination
	Page    int
	PerPage int
	Offset  int

	// Filtering
	Status   string // Order status: "pending", "processing", "on-hold", "completed", "cancelled", "refunded", "failed", "trash"
	Customer string // Customer ID
	Product  string // Product ID
	Search   string // Search for orders by order number or customer details
	After    string // Filter orders created after this date (ISO8601 format)
	Before   string // Filter orders created before this date (ISO8601 format)
	Modified string // Filter orders modified after this date (ISO8601 format)

	// Sorting
	OrderBy string // Sort by: "date", "id", "include", "title", "slug"
	Order   string // Sort order: "asc", "desc"

	// Include/Exclude specific IDs
	Include string // Comma-separated list of order IDs to include
	Exclude string // Comma-separated list of order IDs to exclude
}

// ListOrders retrieves all orders with optional filtering
func (c *Client) ListOrders(options *OrderListOptions) (*OrdersResponse, error) {
	params := make(map[string]string)
	requestOptions := &RequestOptions{
		Params: params,
	}

	if options != nil {
		// Set pagination
		requestOptions.Page = options.Page
		requestOptions.PerPage = options.PerPage
		requestOptions.Offset = options.Offset

		// Set filtering parameters
		if options.Status != "" {
			params["status"] = options.Status
		}
		if options.Customer != "" {
			params["customer"] = options.Customer
		}
		if options.Product != "" {
			params["product"] = options.Product
		}
		if options.Search != "" {
			params["search"] = options.Search
		}
		if options.After != "" {
			params["after"] = options.After
		}
		if options.Before != "" {
			params["before"] = options.Before
		}
		if options.Modified != "" {
			params["modified_after"] = options.Modified
		}
		if options.OrderBy != "" {
			params["orderby"] = options.OrderBy
		}
		if options.Order != "" {
			params["order"] = options.Order
		}
		if options.Include != "" {
			params["include"] = options.Include
		}
		if options.Exclude != "" {
			params["exclude"] = options.Exclude
		}
	}

	response, err := c.GET("orders", requestOptions)
	if err != nil {
		return nil, err
	}

	// Parse the response data into Order structs
	var orders []Order
	if response.Data != nil {
		jsonData, err := json.Marshal(response.Data)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal response data: %w", err)
		}

		if err := json.Unmarshal(jsonData, &orders); err != nil {
			return nil, fmt.Errorf("failed to unmarshal orders: %w", err)
		}
	}

	return &OrdersResponse{
		Orders:     orders,
		Pagination: response.Pagination,
		Headers:    response.Headers,
	}, nil
}

// GetOrder retrieves a single order by ID
func (c *Client) GetOrder(orderID int) (*Order, error) {
	endpoint := fmt.Sprintf("orders/%d", orderID)
	response, err := c.GET(endpoint, nil)
	if err != nil {
		return nil, err
	}

	var order Order
	if response.Data != nil {
		jsonData, err := json.Marshal(response.Data)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal response data: %w", err)
		}

		if err := json.Unmarshal(jsonData, &order); err != nil {
			return nil, fmt.Errorf("failed to unmarshal order: %w", err)
		}
	}

	return &order, nil
}

// IsSubscriptionOrder checks if an order is related to subscriptions
func (c *Client) IsSubscriptionOrder(order *Order) bool {
	// Method 1: Check if order was created via subscription
	if order.CreatedVia == "subscription" {
		return true
	}

	// Method 2: Check for subscription renewal metadata
	for _, meta := range order.MetaData {
		if meta.Key == "_subscription_renewal" {
			return true
		}
	}

	// Method 3: Check line items for subscription scheme metadata
	for _, lineItem := range order.LineItems {
		for _, meta := range lineItem.MetaData {
			if meta.Key == "_wcsatt_scheme" {
				return true
			}
		}
	}

	return false
}

// GetSubscriptionRenewalID extracts the subscription ID from renewal orders
func (c *Client) GetSubscriptionRenewalID(order *Order) (int, bool) {
	for _, meta := range order.MetaData {
		if meta.Key == "_subscription_renewal" {
			// Try to convert the value to int
			switch v := meta.Value.(type) {
			case int:
				return v, true
			case float64:
				return int(v), true
			case string:
				// Try to parse string to int if needed
				if id := parseIntFromString(v); id > 0 {
					return id, true
				}
			}
		}
	}
	return 0, false
}

// GetSubscriptionScheme extracts subscription scheme from line items
func (c *Client) GetSubscriptionScheme(order *Order) string {
	for _, lineItem := range order.LineItems {
		for _, meta := range lineItem.MetaData {
			if meta.Key == "_wcsatt_scheme" {
				if scheme, ok := meta.Value.(string); ok {
					return scheme
				}
			}
		}
	}
	return ""
}

// Helper function to parse int from string
func parseIntFromString(s string) int {
	// Simple integer parsing - you might want to use strconv.Atoi for more robust parsing
	var result int
	fmt.Sscanf(s, "%d", &result)
	return result
}

// ListSubscriptionOrders retrieves only subscription-related orders
func (c *Client) ListSubscriptionOrders(options *OrderListOptions) (*OrdersResponse, error) {
	// Get all orders first
	ordersResponse, err := c.ListOrders(options)
	if err != nil {
		return nil, err
	}

	// Filter to only subscription orders
	var subscriptionOrders []Order
	for _, order := range ordersResponse.Orders {
		if c.IsSubscriptionOrder(&order) {
			subscriptionOrders = append(subscriptionOrders, order)
		}
	}

	return &OrdersResponse{
		Orders:     subscriptionOrders,
		Pagination: ordersResponse.Pagination, // Note: pagination will be off since we filtered
		Headers:    ordersResponse.Headers,
	}, nil
}

// ListSubscriptionRenewals retrieves only subscription renewal orders
func (c *Client) ListSubscriptionRenewals(options *OrderListOptions) (*OrdersResponse, error) {
	// Get all orders first
	ordersResponse, err := c.ListOrders(options)
	if err != nil {
		return nil, err
	}

	// Filter to only renewal orders
	var renewalOrders []Order
	for _, order := range ordersResponse.Orders {
		if order.CreatedVia == "subscription" {
			renewalOrders = append(renewalOrders, order)
		}
	}

	return &OrdersResponse{
		Orders:     renewalOrders,
		Pagination: ordersResponse.Pagination,
		Headers:    ordersResponse.Headers,
	}, nil
}

// GetLastOrders retrieves the most recent orders, sorted by date ascending
func (c *Client) GetLastOrders(count int) (*OrdersResponse, error) {
	options := &OrderListOptions{
		Page:    1,
		PerPage: count,
		OrderBy: "date",
		Order:   "desc",
	}

	return c.ListOrders(options)
}

// GetLast10Orders is a convenience method to get the last 10 orders
func (c *Client) GetLast10Orders() (*OrdersResponse, error) {
	return c.GetLastOrders(10)
}