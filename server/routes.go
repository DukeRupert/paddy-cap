package server

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"sort"
	"sync"

	"github.com/dukerupert/paddy-cap/service/order"
)

func addRoutes(l *slog.Logger, m *http.ServeMux, t *TemplateRenderer, o *order.OrderService) {
	m.Handle("GET /", handleHome(t))
	m.Handle("GET /healthz", handleHealthZ())
	m.Handle("GET /api/orders", handleGetOrders(l, o))
}

func handleHome(t *TemplateRenderer) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		data := map[string]interface{}{
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

func handleGetOrders(l *slog.Logger, o *order.OrderService) http.Handler {
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

		err := encode(w, r, int(http.StatusOK), orders)
		if err != nil {
			l.Error("handleGetAsyncOrders failed", "error_message", err)
			http.Error(w, "Failed to retrieve orders", http.StatusInternalServerError)
		}
	})
}
