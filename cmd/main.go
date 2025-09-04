package main

import (
	"log"
	"log/slog"
	"net"
	"net/http"
	"os"
	"time"

	"github.com/dukerupert/paddy-cap/server"
	"github.com/dukerupert/paddy-cap/service/order"

	_ "github.com/joho/godotenv/autoload"
)

type Config struct {
	// App
	Host string
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

func GetEnv() Config {
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

	if orderspaceBaseURL == "" || orderspaceClientID == "" || orderspaceClientSecret == "" {
		log.Fatal("Missing orderspace environment variables")
	}

	wooBaseURL := os.Getenv("WOO_BASE_URL")
	wooConsumerKey := os.Getenv("WOO_CONSUMER_KEY")
	wooConsumerSecret := os.Getenv("WOO_CONSUMER_SECRET")

	if wooBaseURL == "" || wooConsumerKey == "" || wooConsumerSecret == "" {
		log.Fatal("Missing woocommerce environment variables")
	}

	dbConnectionString := os.Getenv("DB_CONNECTION_STRING")

	return Config{
		Host:					host,
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

func main() {
	// getEnv
	cfg := GetEnv()

	// Setup logger
	logger := slog.New(slog.NewTextHandler(os.Stderr, nil))
	slog.SetDefault(logger)

	// Init Services
	orderService := order.New(logger, order.OrderServiceConfig{
		WooBaseURL: cfg.WooBaseURL,
		WooConsumerKey: cfg.WooConsumerKey,
		WooConsumerSecret: cfg.WooConsumerSecret,
		OrderspaceBaseURL: cfg.OrderspaceBaseURL,
		OrderspaceClientID: cfg.OrderspaceClientID,
		OrderspaceClientSecret: cfg.OrderspaceClientSecret,
	})

	// Init server handler
	srv := server.New(logger, server.ServerConfig{
		Host: cfg.Host,
		Port: cfg.Port,
	}, orderService)

	// Start server
	s := &http.Server{
		Addr:           net.JoinHostPort(cfg.Host, cfg.Port),
		Handler:        srv,
		ReadTimeout:    10 * time.Second,
		WriteTimeout:   10 * time.Second,
		MaxHeaderBytes: 1 << 20,
	}

	log.Fatal(s.ListenAndServe())
}
