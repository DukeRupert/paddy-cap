package main

import (
	"encoding/json"
	"log"
	"log/slog"
	"net/http"
	// "net/url"
	"os"
	"strconv"
	"time"

	"github.com/dukerupert/paddy-cap/middleware"
	"github.com/dukerupert/paddy-cap/orderspace"
)

func handleHome(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		http.Error(w, "Not found", http.StatusNotFound)
		return
	}
	json.NewEncoder(w).Encode(map[string]string{"status": "healthy"})
}

type AppConfig struct {
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

func getEnv() AppConfig {
	port := os.Getenv("PORT")
	// handle missing
	if port == "" {
		slog.Warn("Missing port, using default 8080")
		port = "8080"
	}

	// check for valid integer
	int, err := strconv.Atoi(port)
	if err != nil {
		slog.Warn("Port must be an integer")
		port = "8080"
	}

	// check for valid range
	ok := isValidPort(int)
	if !ok {
		slog.Warn("Invalid port value. Must be between 0 - 65535")
		port = "8080"
	}

	port = ":" + port

	orderspaceBaseURL := os.Getenv("ORDERSPACE_BASE_URL")
	orderspaceClientID := os.Getenv("ORDERSPACE_CLIENT_ID")
	orderspaceClientSecret := os.Getenv("ORDERSPACE_CLIENT_SECRET")

	wooBaseURL := os.Getenv("WOO_BASE_URL")
	wooConsumerKey := os.Getenv("WOO_CONSUMER_KEY")
	wooConsumerSecret := os.Getenv("WOO_CONSUMER_SECRET")

	dbConnectionString := os.Getenv("DB_CONNECTION_STRING")

	return AppConfig{
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

func isValidPort(port int) bool {
	return port >= 0 && port <= 65535
}

type App struct {
	Orderspace *orderspace.Client
}

func NewApp(orderspaceClient *orderspace.Client) *App {
	return &App{
		Orderspace: orderspaceClient,
	}
}

type User struct {
	ID      int      `json:"id"`
	Name    string   `json:"name"`
	Email   string   `json:"email"`
	Tags    []string `json:"tags"`
	Active  bool     `json:"active"`
	Created int64    `json:"created"`
}

func (a *App) handleGetOrders(w http.ResponseWriter, r *http.Request) {
	logger, _ := r.Context().Value(middleware.LoggerKey).(*slog.Logger)
	orders, err := a.Orderspace.GetLast10Orders()
	if err != nil {
		logger.Error("fetching orders failed", "error_message", err)
		http.Error(w, "failed to fetch order records", http.StatusInternalServerError)
	}
	w.Header().Add("Content-Type", "application/json")
	json.NewEncoder(w).Encode(orders)
}

func main() {
	// getEnv
	cfg := getEnv()

	// Setup logger
	logger := slog.New(slog.NewTextHandler(os.Stderr, nil))
	slog.SetDefault(logger)

	// Start services
	orderspaceClient := orderspace.NewClient(cfg.OrderspaceBaseURL, cfg.OrderspaceClientID, cfg.OrderspaceClientSecret)

	app := NewApp(orderspaceClient)

	mux := http.NewServeMux()
	mux.HandleFunc("GET /", handleHome)
	mux.HandleFunc("GET /orders", app.handleGetOrders)

	stack := middleware.CreateStack(middleware.RequestID, middleware.CORS, middleware.Logging)

	s := &http.Server{
		Addr:           cfg.Port,
		Handler:        stack(mux),
		ReadTimeout:    10 * time.Second,
		WriteTimeout:   10 * time.Second,
		MaxHeaderBytes: 1 << 20,
	}
	log.Fatal(s.ListenAndServe())
}
