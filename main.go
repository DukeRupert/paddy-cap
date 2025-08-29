package main

import (
	"encoding/json"
	"log"
	"log/slog"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/dukerupert/paddy-cap/middleware"
)

func handleHome(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		http.Error(w, "Not found", http.StatusNotFound)
		return
	}
	json.NewEncoder(w).Encode(map[string]string{"Status": "healthy"})
}

type AppConfig struct {
	Port string
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

	return AppConfig{
		Port: port,
	}
}

func isValidPort(port int) bool {
	return port >= 0 && port <= 65535
}


func main() {
	// getEnv
	cfg := getEnv()
	// Setup logger
	logger := slog.New(slog.NewTextHandler(os.Stderr, nil))
	slog.SetDefault(logger)

	mux := http.NewServeMux()
	mux.HandleFunc("GET /", handleHome)

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
