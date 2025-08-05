package Yeastar

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
	"go.uber.org/zap"
)

// YeastarIntegration handles the integration with Yeastar
type YeastarIntegration struct {
	log          *zap.Logger
	cortezaURL   string
	eventMonitor *EventMonitor
}

// NewYeastarIntegration creates a new Yeastar integration
func NewYeastarIntegration(log *zap.Logger, cortezaURL string) *YeastarIntegration {
	log.Info("Creating new Yeastar integration", zap.String("cortezaURL", cortezaURL))
	return &YeastarIntegration{
		log:        log,
		cortezaURL: cortezaURL,
	}
}

// Start initializes and starts the Yeastar integration
func (yi *YeastarIntegration) Start(ctx context.Context, host string) error {
	yi.log.Info("Starting Yeastar integration")

	// Initialize global managers
	yi.log.Info("Initializing global managers")
	InitializeGlobalManagers()

	// Create Corteza client
	yi.log.Info("Creating Corteza client")
	url := "http://" + host
	cortezaClient := NewCortezaClient(url)

	// Create event monitor
	yi.log.Info("Initializing event monitor")
	yi.eventMonitor = NewEventMonitor(
		GlobalConfigManager,
		GlobalTokenManager,
		cortezaClient,
	)

	// Start event monitoring in a separate goroutine
	go func() {
		yi.log.Info("Starting event monitor goroutine")
		if err := yi.eventMonitor.Start(ctx); err != nil {
			yi.log.Error("Failed to start event monitor", zap.Error(err))
		}
	}()

	yi.log.Info("Yeastar integration started successfully")
	return nil
}

// Middleware returns the middleware for handling Yeastar routes
func (yi *YeastarIntegration) Middleware() func(chi.Router) {
	return func(r chi.Router) {
		r.Route("/yeastar", func(r chi.Router) {
			// Sync all data
			r.Get("/sync", HandleSyncAllHTTP)

			// Event monitor status
			r.Get("/events/status", yi.handleEventStatus)

			// Start/stop event monitoring
			r.Post("/events/start", yi.handleStartEvents)
			r.Post("/events/stop", yi.handleStopEvents)
		})
	}
}

// handleEventStatus returns the status of event monitoring
func (yi *YeastarIntegration) handleEventStatus(w http.ResponseWriter, r *http.Request) {
	yi.log.Info("Handling request: GET /yeastar/events/status")

	status := map[string]interface{}{
		"running":   false,
		"connected": false,
	}

	if yi.eventMonitor != nil {
		isRunning := yi.eventMonitor.IsRunning()
		status["running"] = isRunning
		status["connected"] = isRunning // Simplified assumption
	} else {
		yi.log.Warn("Event monitor is nil in status check")
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	if err := json.NewEncoder(w).Encode(status); err != nil {
		yi.log.Error("Failed to encode status response", zap.Error(err))
	}
}

// handleStartEvents starts event monitoring
func (yi *YeastarIntegration) handleStartEvents(w http.ResponseWriter, r *http.Request) {
	yi.log.Info("Handling request: POST /yeastar/events/start")

	if yi.eventMonitor == nil {
		yi.log.Error("Start failed: event monitor not initialized")
		http.Error(w, "Event monitor not initialized", http.StatusInternalServerError)
		return
	}

	if yi.eventMonitor.IsRunning() {
		yi.log.Warn("Start requested but event monitor is already running")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("Event monitoring is already running"))
		return
	}

	go func() {
		yi.log.Info("Starting event monitor...")
		if err := yi.eventMonitor.Start(r.Context()); err != nil {
			yi.log.Error("Failed to start event monitoring", zap.Error(err))
		} else {
			yi.log.Info("Event monitoring started successfully")
		}
	}()

	w.WriteHeader(http.StatusOK)
	w.Write([]byte("Event monitoring started"))
}

// handleStopEvents stops event monitoring
func (yi *YeastarIntegration) handleStopEvents(w http.ResponseWriter, r *http.Request) {
	yi.log.Info("Handling request: POST /yeastar/events/stop")

	if yi.eventMonitor == nil {
		yi.log.Error("Stop failed: event monitor not initialized")
		http.Error(w, "Event monitor not initialized", http.StatusInternalServerError)
		return
	}

	if err := yi.eventMonitor.Stop(); err != nil {
		yi.log.Error("Failed to stop event monitoring", zap.Error(err))
		http.Error(w, "Failed to stop event monitoring", http.StatusInternalServerError)
		return
	}

	yi.log.Info("Event monitoring stopped successfully")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("Event monitoring stopped"))
}
