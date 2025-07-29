package Yeastar

import (
	"context"
	"fmt"
	"log"
	"time"
)

// EventMonitor manages the WebSocket connection and event monitoring
type EventMonitor struct {
	yeastarService   *YeastarService
	webSocketService *WebSocketService
	isRunning        bool
}

// NewEventMonitor creates a new event monitor
func NewEventMonitor(configManager *ConfigManager, tokenManager *TokenManager, cortezaClient *CortezaClient) *EventMonitor {
	log.Println("[EventMonitor] Initializing Yeastar and WebSocket services")

	yeastarService := NewYeastarService(configManager, tokenManager, cortezaClient)
	webSocketService := NewWebSocketService(configManager, tokenManager, cortezaClient)

	return &EventMonitor{
		yeastarService:   yeastarService,
		webSocketService: webSocketService,
	}
}

// Add this method to provide access to the event monitor
func (yi *YeastarIntegration) GetEventMonitor() *EventMonitor {
	return yi.eventMonitor
}

// Enhanced EventMonitor with proper initialization flow
func (em *EventMonitor) Start(ctx context.Context) error {
	if em.isRunning {
		log.Println("[EventMonitor] Start requested but monitor is already running")
		return fmt.Errorf("event monitor is already running")
	}

	log.Println("[EventMonitor] Starting event monitor...")

	// This will now trigger config and token pushes from Corteza
	log.Println("[EventMonitor] Initializing Yeastar service (will trigger Corteza pushes)...")
	if err := em.yeastarService.WaitForInitialization(ctx); err != nil {
		return fmt.Errorf("failed to initialize Yeastar service: %w", err)
	}

	log.Println("[EventMonitor] Ensuring valid token...")
	if err := em.yeastarService.EnsureValidToken(ctx); err != nil {
		return fmt.Errorf("failed to ensure valid token: %w", err)
	}

	log.Println("[EventMonitor] Connecting to Yeastar WebSocket...")
	if err := em.webSocketService.Connect(ctx); err != nil {
		return fmt.Errorf("failed to connect to WebSocket: %w", err)
	}

	eventIDs := []int{
		EventExtensionRegistration,
		EventExtensionCallStatus,
		EventExtensionPresenceStatus,
		EventCallStatusChanged,
		EventNewCDR,
		EventCallTransfer,
		EventCallFoward,
		EventCallStatus,
		EventSatisfaction,
		EventUaCSTACall,
		EventExtensionConfiguration,
		EventAgentPause,
		EventAgentRingTimeout,
		EventReportDownload,
		EventCallNoteStatusChanged,
		EventAgentStatusChanged,
	}

	log.Printf("[EventMonitor] Subscribing to event topics: %v", eventIDs)
	if err := em.webSocketService.Subscribe(eventIDs); err != nil {
		return fmt.Errorf("failed to subscribe to events: %w", err)
	}

	log.Println("[EventMonitor] Starting heartbeat...")
	em.webSocketService.StartHeartbeat()

	em.isRunning = true

	// Start listening for events in a separate goroutine
	go func() {
		defer func() {
			log.Println("[EventMonitor] Event listener stopped, cleaning up")
			em.isRunning = false
			em.webSocketService.Close()
		}()

		for {
			log.Println("[EventMonitor] Listening for WebSocket events...")
			if err := em.webSocketService.Listen(ctx); err != nil {
				log.Printf("[EventMonitor] WebSocket listener error: %v", err)
				log.Println("[EventMonitor] Attempting to reconnect in 30 seconds...")
				time.Sleep(30 * time.Second)

				select {
				case <-ctx.Done():
					log.Println("[EventMonitor] Context canceled, stopping reconnection attempts")
					return
				default:
				}

				if err := em.reconnect(ctx); err != nil {
					log.Printf("[EventMonitor] Reconnection failed: %v", err)
					continue
				}
			}
		}
	}()

	log.Println("[EventMonitor] ✅ Yeastar event monitor started successfully")
	return nil
}

// Enhanced reconnect with proper token refresh
func (em *EventMonitor) reconnect(ctx context.Context) error {
	log.Println("[EventMonitor] Reconnecting to Yeastar WebSocket...")

	em.webSocketService.Close()

	log.Println("[EventMonitor] Ensuring valid token for reconnection...")
	if err := em.yeastarService.EnsureValidToken(ctx); err != nil {
		return fmt.Errorf("failed to ensure valid token for reconnection: %w", err)
	}

	log.Println("[EventMonitor] Reconnecting WebSocket...")
	if err := em.webSocketService.Connect(ctx); err != nil {
		return fmt.Errorf("failed to reconnect to WebSocket: %w", err)
	}

	eventIDs := []int{
		EventExtensionRegistration,
		EventExtensionCallStatus,
		EventExtensionPresenceStatus,
		EventCallStatusChanged,
		EventNewCDR,
		EventCallTransfer,
		EventCallFoward,
		EventCallStatus,
		EventSatisfaction,
		EventUaCSTACall,
		EventExtensionConfiguration,
		EventAgentPause,
		EventAgentRingTimeout,
		EventReportDownload,
		EventCallNoteStatusChanged,
		EventAgentStatusChanged,
	}

	log.Printf("[EventMonitor] Resubscribing to event topics: %v", eventIDs)
	if err := em.webSocketService.Subscribe(eventIDs); err != nil {
		return fmt.Errorf("failed to resubscribe to events: %w", err)
	}

	log.Println("[EventMonitor] Restarting heartbeat...")
	em.webSocketService.StartHeartbeat()

	log.Println("[EventMonitor] ✅ Successfully reconnected to Yeastar WebSocket")
	return nil
}

// Stop stops the event monitoring
func (em *EventMonitor) Stop() error {
	if !em.isRunning {
		log.Println("[EventMonitor] Stop requested but monitor is not running")
		return nil
	}

	log.Println("[EventMonitor] Stopping Yeastar event monitor...")

	if err := em.webSocketService.Close(); err != nil {
		log.Printf("[EventMonitor] Error closing WebSocket: %v", err)
	}

	em.isRunning = false
	log.Println("[EventMonitor] Yeastar event monitor stopped successfully")
	return nil
}

// IsRunning returns whether the monitor is running
func (em *EventMonitor) IsRunning() bool {
	return em.isRunning
}
