package yeastarEvents

import (
	"context"
	"encoding/json"
	"log"
	"net/http"

	"github.com/go-chi/chi/v5"
)

// EventReceiver handles incoming events from Yeastar
type EventReceiver struct {
	processor *EventProcessor
}

func NewEventReceiver(processor *EventProcessor) *EventReceiver {
	return &EventReceiver{
		processor: processor,
	}
}

// RegisterRoutes adds Yeastar webhook routes to your existing chi router
func (er *EventReceiver) RegisterRoutes(r chi.Router) {
	r.Route("/yeastar", func(r chi.Router) {
		r.Post("/webhook", er.handleWebhook)
		r.Get("/health", er.handleHealth)
	})
}

// Start now only starts the WebSocket client (no HTTP server)
func (er *EventReceiver) Start(ctx context.Context) error {
	log.Println("Starting Yeastar Event Receiver...")

	// Start WebSocket client in background
	go er.startWebSocketClient(ctx)

	return nil
}

func (er *EventReceiver) handleWebhook(w http.ResponseWriter, r *http.Request) {
	var event map[string]interface{}
	if err := json.NewDecoder(r.Body).Decode(&event); err != nil {
		log.Printf("Failed to decode Yeastar webhook: %v", err)
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	log.Printf("Received Yeastar webhook event: %v", event)

	// Send to processor (non-blocking)
	er.processor.ProcessEvent(event)

	// Respond OK to Yeastar
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(`{"status":"ok"}`))
}

func (er *EventReceiver) handleHealth(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(`{"status":"healthy","service":"yeastar-integration"}`))
}

func (er *EventReceiver) startWebSocketClient(ctx context.Context) {
	// WebSocket client implementation would go here
	log.Println("WebSocket client started (placeholder)")

	// Wait for context cancellation
	<-ctx.Done()
	log.Println("WebSocket client stopped")
}
