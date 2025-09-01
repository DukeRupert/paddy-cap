package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"log/slog"
	"net"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"sync"

	"os"
	"time"

	"github.com/dukerupert/paddy-cap/middleware"
	"github.com/dukerupert/paddy-cap/orderspace"
	"github.com/dukerupert/paddy-cap/woocommerce"
	"golang.org/x/text/cases"
	"golang.org/x/text/language"
)

type ServerConfig struct {
	// App
	Port string
	// Orderspace Client
	OrderspaceBaseURL      string
	OrderspaceClientID     string
	OrderspaceClientSecret string
	// Woocommerce Client
	WooBaseURL        string
	WooConsumerKey    string
	WooConsumerSecret string
	// Database
	ConnectionString string
}

func getEnv() ServerConfig {
	host := os.Getenv("HOST")
	if host == "" {
		host = "localhost"
	}
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	orderspaceBaseURL := os.Getenv("ORDERSPACE_BASE_URL")
	orderspaceClientID := os.Getenv("ORDERSPACE_CLIENT_ID")
	orderspaceClientSecret := os.Getenv("ORDERSPACE_CLIENT_SECRET")

	wooBaseURL := os.Getenv("WOO_BASE_URL")
	wooConsumerKey := os.Getenv("WOO_CONSUMER_KEY")
	wooConsumerSecret := os.Getenv("WOO_CONSUMER_SECRET")

	dbConnectionString := os.Getenv("DB_CONNECTION_STRING")

	return ServerConfig{
		Port:                   port,
		OrderspaceBaseURL:      orderspaceBaseURL,
		OrderspaceClientID:     orderspaceClientID,
		OrderspaceClientSecret: orderspaceClientSecret,
		WooBaseURL:             wooBaseURL,
		WooConsumerKey:         wooConsumerKey,
		WooConsumerSecret:      wooConsumerSecret,
		ConnectionString:       dbConnectionString,
	}
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
	wooClient        *woocommerce.Client
	orderspaceClient *orderspace.Client
	titleCaser       cases.Caser
}

func NewOrderService(logger *slog.Logger, cfg ServerConfig) *OrderService {
	orderspaceClient := orderspace.NewClient(cfg.OrderspaceBaseURL, cfg.OrderspaceClientID, cfg.OrderspaceClientSecret)
	woocommerceClient := woocommerce.NewClient(cfg.WooBaseURL, cfg.WooConsumerKey, cfg.WooConsumerSecret)
	// Create a title caser for English
	titleCaser := cases.Title(language.English)

	service := &OrderService{
		wooClient:        woocommerceClient,
		orderspaceClient: orderspaceClient,
		titleCaser:       titleCaser,
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
		Status:      s.titleCaser.String(order.Status),
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
		Status:      s.titleCaser.String(order.Status),
		Origin:      "Orderspace",
		SortDate:    sortDate,
	}
}

func NewServer(logger *slog.Logger, cfg ServerConfig) http.Handler {
	mux := http.NewServeMux()
	orderService := NewOrderService(logger, cfg)
	addRoutes(logger, mux, orderService)
	var handler http.Handler = mux
	// Middleware here
	handler = middleware.Logging(handler)
	handler = middleware.RequestID(handler)
	handler = middleware.CORS(handler)
	return handler
}

func addRoutes(logger *slog.Logger, mux *http.ServeMux, orderService *OrderService) {
	mux.HandleFunc("GET /", handleHome)
	mux.Handle("GET /api/orders", handleGetOrders(logger, orderService))
}

func handleHome(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		http.Error(w, "Not found", http.StatusNotFound)
		return
	}
	json.NewEncoder(w).Encode(map[string]string{"status": "healthy"})
}

func handleGetOrders(logger *slog.Logger, orderService *OrderService) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var wg sync.WaitGroup
		var mu sync.Mutex
		orders := []Order{}

		// Fetch and transform orders
		wg.Add(1)
		go func() {
			defer wg.Done()
			res, err := orderService.orderspaceClient.GetLast10Orders()
			if err != nil {
				logger.Error("fetching orderspace orders failed", "error_message", err)
			}
			transformedOrders := []Order{}
			for _, v := range res.Orders {
				o := orderService.ConvertOrderspaceOrder(v)
				transformedOrders = append(transformedOrders, o)
			}

			for _, v := range transformedOrders {
				mu.Lock()
				orders = append(orders, v)
				mu.Unlock()
			}
		}()

		wg.Add(1)
		go func() {
			defer wg.Done()
			res, err := orderService.wooClient.GetLast10Orders()
			if err != nil {
				logger.Error("fetching woocommerce orders failed", "error_message", err)
			}
			transformedOrders := []Order{}
			for _, v := range res.Orders {
				o := orderService.ConvertWooOrder(v)
				transformedOrders = append(transformedOrders, o)
			}

			for _, v := range transformedOrders {
				mu.Lock()
				orders = append(orders, v)
				mu.Unlock()
			}
		}()

		wg.Wait()

		// Sort
		sort.Slice(orders, func(i, j int) bool {
			return orders[i].SortDate.After(orders[j].SortDate) // descending
		})

		err := encode(w, r, int(http.StatusOK), orders)
		if err != nil {
			logger.Error("handleGetAsyncOrders failed", "error_message", err)
			http.Error(w, "Failed to retrieve orders", http.StatusInternalServerError)
		}
	})
}

func encode[T any](w http.ResponseWriter, r *http.Request, status int, v T) error {
	w.Header().Set("Content-Type", "application/json")
	if status != 200 {
		w.WriteHeader(status)
	}
	if err := json.NewEncoder(w).Encode(v); err != nil {
		return fmt.Errorf("encode json: %w", err)
	}
	return nil
}

func decode[T any](r *http.Request) (T, error) {
	var v T
	if err := json.NewDecoder(r.Body).Decode(&v); err != nil {
		return v, fmt.Errorf("decode json: %w", err)
	}
	return v, nil
}

// Validator is an object that can be validated.
type Validator interface {
	// Valid checks the object and returns any
	// problems. If len(problems) == 0 then
	// the object is valid.
	Valid(ctx context.Context) (problems map[string]string)
}

func decodeValid[T Validator](r *http.Request) (T, map[string]string, error) {
	var v T
	if err := json.NewDecoder(r.Body).Decode(&v); err != nil {
		return v, nil, fmt.Errorf("decode json: %w", err)
	}
	if problems := v.Valid(r.Context()); len(problems) > 0 {
		return v, problems, fmt.Errorf("invalid %T: %d problems", v, len(problems))
	}
	return v, nil, nil
}

func main() {
	// getEnv
	cfg := getEnv()

	// Setup logger
	logger := slog.New(slog.NewTextHandler(os.Stderr, nil))
	slog.SetDefault(logger)

	srv := NewServer(logger, cfg)

	s := &http.Server{
		Addr:           net.JoinHostPort("localhost", cfg.Port),
		Handler:        srv,
		ReadTimeout:    10 * time.Second,
		WriteTimeout:   10 * time.Second,
		MaxHeaderBytes: 1 << 20,
	}

	log.Fatal(s.ListenAndServe())
}
