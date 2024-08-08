package handler

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"order-event-processor/internal/broadcaster"
	"order-event-processor/internal/model"
	"time"
)

type OrderEventStreamHandler struct {
	log         *slog.Logger
	broadcaster *broadcaster.OrderEventBroadcaster
}

func NewOrderEventStreamHandler(log *slog.Logger, b *broadcaster.OrderEventBroadcaster) *OrderEventStreamHandler {
	return &OrderEventStreamHandler{
		log:         log,
		broadcaster: b,
	}
}

func (h *OrderEventStreamHandler) StreamOrderEvents(w http.ResponseWriter, r *http.Request) {
	orderId := r.PathValue("order_id")

	// Set headers for SSE
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.(http.Flusher).Flush()

	channel := make(chan *model.OrderEvent)
	h.broadcaster.RegisterChannel(orderId, channel)

	for {
		select {
		case event, ok := <-channel:
			if !ok {
				return
			}
			jsonData, err := json.Marshal(event)
			if err != nil {
				return
			}
			fmt.Fprintf(w, "data: %s\n\n", string(jsonData))
			w.(http.Flusher).Flush()
		case <-time.After(1 * time.Minute):
			h.broadcaster.UnregisterChannel(orderId, channel)
			return
		}
	}
}
