package server

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"sort"
	"strconv"
	"sync"

	"github.com/dukerupert/paddy-cap/service/order"
)

func addRoutes(l *slog.Logger, m *http.ServeMux, t *TemplateRenderer, o *order.OrderService) {
	m.Handle("GET /", handleHome(t))
	m.Handle("GET /healthz", handleHealthZ())
	m.Handle("GET /orders", handleGetOrders(l, t, o))
	m.Handle("GET /orders/{origin}/{id}", handleGetOrder(l, t, o))

}

func handleHome(t *TemplateRenderer) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		data := map[string]any{
			"Title":   "Home Page",
			"Message": "Welcome to our website!",
			"User": map[string]string{
				"Name":  "John Doe",
				"Email": "john@example.com",
			},
		}

		if err := t.Render(w, "home", data); err != nil {
			http.Error(w, "Error rendering template: "+err.Error(), http.StatusInternalServerError)
			return
		}
	})
}

func handleHealthZ() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/" {
			http.Error(w, "Not found", http.StatusNotFound)
			return
		}
		json.NewEncoder(w).Encode(map[string]string{"status": "healthy"})
	})
}

func handleGetOrders(l *slog.Logger, t *TemplateRenderer, o *order.OrderService) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var wg sync.WaitGroup
		var mu sync.Mutex
		orders := []order.Order{}

		// Fetch and transform orders
		wg.Go(func() {
			res, err := o.OrderspaceClient.GetLast10Orders()
			if err != nil {
				l.Error("fetching orderspace orders failed", "error_message", err)
			}
			transformed := []order.Order{}
			for _, v := range res.Orders {
				o := o.ConvertOrderspaceOrder(v)
				transformed = append(transformed, o)
			}

			for _, v := range transformed {
				mu.Lock()
				orders = append(orders, v)
				mu.Unlock()
			}
		})

		wg.Go(func() {
			res, err := o.WooClient.GetLast10Orders()
			if err != nil {
				l.Error("fetching woocommerce orders failed", "error_message", err)
			}
			transformed := []order.Order{}
			for _, v := range res.Orders {
				o := o.ConvertWooOrder(v)
				transformed = append(transformed, o)
			}

			for _, v := range transformed {
				mu.Lock()
				orders = append(orders, v)
				mu.Unlock()
			}
		})

		wg.Wait()

		// Sort
		sort.Slice(orders, func(i, j int) bool {
			return orders[i].SortDate.After(orders[j].SortDate) // descending
		})

		// Check content type header and return json or html
		// contentType := getContentType(l, r)
		if r.Header.Get("Content-Type") == "application/json" {
			err := encode(w, r, int(http.StatusOK), orders)
			if err != nil {
				l.Error("handleGetAsyncOrders failed", "error_message", err)
				http.Error(w, "Failed to retrieve orders", http.StatusInternalServerError)
			}
			return
		}

		// case html
		data := map[string]any{
			"Title":  "Orders Page",
			"Orders": orders,
		}
		if err := t.Render(w, "orders", data); err != nil {
			http.Error(w, "Error rendering template: "+err.Error(), http.StatusInternalServerError)
			return
		}
	})
}

func handleGetOrder(l *slog.Logger, t *TemplateRenderer, o *order.OrderService) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		orderID := r.PathValue("id")
		origin := r.PathValue("origin")
		if origin == "" || !validateOrigin(origin) {
			l.Warn("Invalid or missing origin", "origin", origin)
			http.Error(w, "invalid or missing origin", http.StatusBadRequest)
			return
		}
		l.Info("Retrieve single order", "orderID", orderID, "origin", origin)
		switch origin {
		case Orderspace:
			order, err := o.OrderspaceClient.GetOrder(orderID)
			if err != nil {
				l.Error("error retrieving order details", "error_message", err.Error(), "orderID", orderID, "origin", origin)
				http.Error(w, "failed to retrieve order details", http.StatusInternalServerError)
				return
			}
			data := map[string]any{
				"Title": "Orders Page",
				"Order": order,
			}
			if err := t.Render(w, "order-details-orderspace", data); err != nil {
				http.Error(w, "Error rendering template: "+err.Error(), http.StatusInternalServerError)
				return
			}
			return
			// err = encode(w, r, http.StatusOK, order)
			// if err != nil {
			// 	l.Error("failed to encode order details", "error_message", err.Error())
			// 	http.Error(w, "failed to encode order details", http.StatusInternalServerError)
			// 	return
			// }
			// return
		case WooCommerce:
			oid, err := strconv.Atoi(orderID)
			if err != nil {
				l.Error("error parsing woocommerce orderID", "error_message", err.Error(), "orderID", orderID)
				http.Error(w, "invalid orderID", http.StatusBadRequest)
				return
			}
			order, err := o.WooClient.GetOrder(oid)
			if err != nil {
				l.Error("error retrieving order details", "error_message", err.Error(), "orderID", orderID, "origin", origin)
				http.Error(w, "failed to retrieve order details", http.StatusInternalServerError)
				return
			}
			err = encode(w, r, http.StatusOK, order)
			if err != nil {
				l.Error("failed to encode order details", "error_message", err.Error())
				http.Error(w, "failed to encode order details", http.StatusInternalServerError)
				return
			}
			return
		default:
			json.NewEncoder(w).Encode(map[string]string{"status": "healthy", "orderID": orderID, "origin": origin})
		}

	})
}

func validateOrigin(origin string) bool {
	if origin == WooCommerce || origin == Orderspace {
		return true
	}
	return false
}

func getContentType(l *slog.Logger, r *http.Request) string {
	// var content string
	header := r.Header.Get("Content-Type")
	l.Info("getContentType()", "Content-Type", header)
	return header
}
