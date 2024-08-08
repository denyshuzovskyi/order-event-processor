package handler

import (
	"log/slog"
	"net/http"
	"order-event-processor/internal/lib/httputil"
	"order-event-processor/internal/model"
)

type OrdersFinder interface {
	GetAllOrders() ([]model.Order, error)
}

type OrdersHandler struct {
	log    *slog.Logger
	finder OrdersFinder
}

func NewOrdersHandler(log *slog.Logger, finder OrdersFinder) *OrdersHandler {
	return &OrdersHandler{
		log:    log,
		finder: finder,
	}
}

func (h *OrdersHandler) GetAllOrders(w http.ResponseWriter, _ *http.Request) {
	orders, err := h.finder.GetAllOrders()
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		h.log.Error("error while getting broadcaster", "error", err)
		return
	}

	if err = httputil.WriteJSON(w, orders); err != nil {
		h.log.Error("error while writing orders", "error", err)
		return
	}
}
