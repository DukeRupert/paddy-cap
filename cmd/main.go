package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"log/slog"
	"net"
	"net/http"

	"os"
	"time"

	"github.com/dukerupert/paddy-cap/middleware"
	"github.com/dukerupert/paddy-cap/orderspace"
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

func NewServer(logger *slog.Logger, cfg ServerConfig) http.Handler {
	mux := http.NewServeMux()
	// Start services
	orderspaceClient := orderspace.NewClient(cfg.OrderspaceBaseURL, cfg.OrderspaceClientID, cfg.OrderspaceClientSecret)
	addRoutes(logger, mux, orderspaceClient)
	var handler http.Handler = mux
	// Middleware here
	handler = middleware.Logging(handler)
	handler = middleware.RequestID(handler)
	handler = middleware.CORS(handler)
	return handler
}

func addRoutes(logger *slog.Logger, mux *http.ServeMux, orderspaceClient *orderspace.Client) {
	mux.HandleFunc("GET /", handleHome)
	mux.Handle("GET /orders", handleGetOrders(logger, orderspaceClient))
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

func handleHome(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		http.Error(w, "Not found", http.StatusNotFound)
		return
	}
	json.NewEncoder(w).Encode(map[string]string{"status": "healthy"})
}

func handleGetOrders(logger *slog.Logger, orderspaceClient *orderspace.Client) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		orders, err := orderspaceClient.GetLast10Orders()
		if err != nil {
			logger.Error("fetching orders failed", "error_message", err)
			http.Error(w, "failed to fetch order records", http.StatusInternalServerError)
		}
		// w.Header().Add("Content-Type", "application/json")
		// json.NewEncoder(w).Encode(orders)
		err = encode(w, r, int(http.StatusOK), orders)
	})

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
