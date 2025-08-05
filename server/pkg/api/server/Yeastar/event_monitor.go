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

func (em *EventMonitor) Start(ctx context.Context) error {
	if em.isRunning {
		log.Println("[EventMonitor] Start requested but monitor is already running")
		return fmt.Errorf("event monitor is already running")
	}

	log.Println("[EventMonitor] ✅ Starting Yeastar event monitor (with automatic retries)...")
	em.isRunning = true

	go em.runMonitorLoop(ctx)

	return nil
}

func (em *EventMonitor) runMonitorLoop(ctx context.Context) {
	defer func() {
		em.isRunning = false
		em.webSocketService.Close()
		log.Println("[EventMonitor] Monitor loop stopped, WebSocket closed")
	}()

	eventIDs := []int{
		EventExtensionRegistration,
		EventExtensionCallStatus,
		EventExtensionPresenceStatus,
		EventCallStatusChanged,
		EventNewCDR,
		EventCallTransfer,
		EventCallFoward,
		EventCallFailed,
		EventSatisfaction,
		EventUaCSTACall,
		EventExtensionConfiguration,
		EventAgentPause,
		EventAgentRingTimeout,
		EventReportDownload,
		EventCallNoteStatusChanged,
		EventAgentStatusChanged,
	}

	for {
		// Cancel loop if context is done
		if ctx.Err() != nil {
			log.Println("[EventMonitor] Context canceled, exiting monitor loop")
			return
		}

		// Step 1: Setup and token
		log.Println("[EventMonitor] Setting up Yeastar service and ensuring token...")
		if err := setupAuth(ctx, em.yeastarService); err != nil {
			log.Printf("[EventMonitor] setupAuth failed: %v", err)
			time.Sleep(30 * time.Second)
			continue
		}

		if err := em.yeastarService.EnsureValidToken(ctx); err != nil {
			log.Printf("[EventMonitor] Failed to get valid token: %v", err)
			time.Sleep(30 * time.Second)
			continue
		}

		// Step 2: WebSocket Connect
		log.Println("[EventMonitor] Connecting to WebSocket...")
		if err := em.webSocketService.Connect(ctx); err != nil {
			log.Printf("[EventMonitor] WebSocket connection failed: %v", err)
			time.Sleep(30 * time.Second)
			continue
		}

		// Step 3: Subscribe
		log.Println("[EventMonitor] Subscribing to events...")
		if err := em.webSocketService.Subscribe(eventIDs); err != nil {
			log.Printf("[EventMonitor] Subscription failed: %v", err)
			em.webSocketService.Close()
			time.Sleep(30 * time.Second)
			continue
		}

		// Step 4: Start heartbeat
		em.webSocketService.StartHeartbeat()

		// Step 5: Listen (blocking)
		log.Println("[EventMonitor] Listening for WebSocket events...")
		err := em.webSocketService.Listen(ctx)
		if err != nil {
			log.Printf("[EventMonitor] Listen error: %v", err)
		}

		// Step 6: Clean up and retry
		em.webSocketService.Close()
		log.Println("[EventMonitor] Disconnected. Retrying in 30 seconds...")
		time.Sleep(30 * time.Second)
	}
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
