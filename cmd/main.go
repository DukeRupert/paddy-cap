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
)

func main() {
	// getEnv
	cfg := getEnv()

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
