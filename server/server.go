package server

import (
	"log/slog"
	"net/http"

	"github.com/dukerupert/paddy-cap/middleware"
	"github.com/dukerupert/paddy-cap/service/order"
)

func New(logger *slog.Logger, cfg ServerConfig, orderService *order.OrderService) http.Handler {
	// Initialize the template renderer
	template, err := NewTemplateRenderer()
	if err != nil {
		panic("Failed to initialize template renderer: " + err.Error())
	}

	mux := http.NewServeMux()
	addRoutes(logger, mux, template, orderService)
	var handler http.Handler = mux
	// Middleware here
	handler = middleware.Logging(handler)
	handler = middleware.RequestID(handler)
	handler = middleware.CORS(handler)
	return handler
}
