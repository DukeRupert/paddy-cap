package order

import (
	"log/slog"
	"strconv"
	"strings"
	"time"

	"github.com/dukerupert/paddy-cap/service/orderspace"
	"github.com/dukerupert/paddy-cap/service/woocommerce"
	"golang.org/x/text/cases"
	"golang.org/x/text/language"
)

type OrderServiceConfig struct {
	// Orderspace Client
	OrderspaceBaseURL      string
	OrderspaceClientID     string
	OrderspaceClientSecret string
	// Woocommerce Client
	WooBaseURL        string
	WooConsumerKey    string
	WooConsumerSecret string
}

// Order represents an order from either system for display
type Order struct {
	ID          string
	OrderNumber int
	Customer    string
	OrderDate   string
	DeliverOn   string
	Total       string
	Status      string
	Origin      string
	SortDate    time.Time // Added for sorting purposes
}

type OrderService struct {
	WooClient        *woocommerce.Client
	OrderspaceClient *orderspace.Client
	TitleCaser       cases.Caser
}

func New(logger *slog.Logger, cfg OrderServiceConfig) *OrderService {
	orderspaceClient := orderspace.NewClient(cfg.OrderspaceBaseURL, cfg.OrderspaceClientID, cfg.OrderspaceClientSecret)
	woocommerceClient := woocommerce.NewClient(cfg.WooBaseURL, cfg.WooConsumerKey, cfg.WooConsumerSecret)
	// Create a title caser for English
	titleCaser := cases.Title(language.English)

	service := &OrderService{
		WooClient:        woocommerceClient,
		OrderspaceClient: orderspaceClient,
		TitleCaser:       titleCaser,
	}

	slog.Info("Order service initialized")
	return service
}

// FormatCurrency formats the currency amount based on the currency
func FormatCurrency(amount float64, currency string) string {
	switch strings.ToUpper(currency) {
	case "USD":
		return "$" + strconv.FormatFloat(amount, 'f', 2, 64)
	case "GBP":
		return "£" + strconv.FormatFloat(amount, 'f', 2, 64)
	case "EUR":
		return "€" + strconv.FormatFloat(amount, 'f', 2, 64)
	default:
		return strconv.FormatFloat(amount, 'f', 2, 64) + " " + currency
	}
}

// ConvertWooOrder converts a WooCommerce order to Order
func (s *OrderService) ConvertWooOrder(order woocommerce.Order) Order {
	customer := strings.TrimSpace(order.Billing.FirstName + " " + order.Billing.LastName)
	if customer == "" {
		customer = order.Billing.Email
	}

	// Parse total
	total, err := strconv.ParseFloat(order.Total, 64)
	if err != nil {
		total = 0
	}

	// Parse date for sorting
	sortDate, err := time.Parse("2006-01-02T15:04:05", order.DateCreated)
	if err != nil {
		slog.Warn("Failed to parse WooCommerce date for sorting", "date", order.DateCreated, "error", err)
		sortDate = time.Now() // Fallback to current time
	}

	// Format date for display
	orderDate := order.DateCreated
	if err == nil {
		orderDate = sortDate.Format("Jan 2, 2006")
	}

	return Order{
		ID:          strconv.Itoa(order.ID),
		OrderNumber: order.ID,
		Customer:    customer,
		OrderDate:   orderDate,
		DeliverOn:   "N/A",
		Total:       FormatCurrency(total, order.Currency),
		Status:      s.TitleCaser.String(order.Status),
		Origin:      "WooCommerce",
		SortDate:    sortDate,
	}
}

// ConvertOrderspaceOrder converts an Orderspace order to UnifiedOrder
func (s *OrderService) ConvertOrderspaceOrder(order orderspace.Order) Order {
	customer := order.CompanyName
	if customer == "" && order.BillingAddress.ContactName != "" {
		customer = order.BillingAddress.ContactName
	}

	// Parse date for sorting
	sortDate, err := time.Parse("2006-01-02T15:04:05Z", order.Created)
	if err != nil {
		slog.Warn("Failed to parse Orderspace date for sorting", "date", order.Created, "error", err)
		sortDate = time.Now() // Fallback to current time
	}

	// Format date for display
	orderDate := order.Created
	if err == nil {
		orderDate = sortDate.Format("Jan 2, 2006")
	}

	deliverOn := "N/A"
	if order.DeliveryDate != "" {
		if parsed, err := time.Parse("2006-01-02", order.DeliveryDate); err == nil {
			deliverOn = parsed.Format("Jan 2, 2006")
		} else {
			deliverOn = order.DeliveryDate
		}
	}

	return Order{
		ID:          order.ID,
		OrderNumber: order.Number,
		Customer:    customer,
		OrderDate:   orderDate,
		DeliverOn:   deliverOn,
		Total:       FormatCurrency(order.GrossTotal, order.Currency),
		Status:      s.TitleCaser.String(order.Status),
		Origin:      "Orderspace",
		SortDate:    sortDate,
	}
}
