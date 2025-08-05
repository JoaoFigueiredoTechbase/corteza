package Yeastar

import (
	"context"
	"errors"
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

var (
	// Add this error definition
	ErrInvalidCredentials = errors.New("invalid credentials")

	// Consider adding other common errors
	ErrNotConnected       = errors.New("websocket not connected")
	ErrSubscriptionFailed = errors.New("subscription failed")
)

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

	const (
		initialBackoff = 5 * time.Second
		maxBackoff     = 5 * time.Minute
		maxAuthRetries = 3
	)

	var (
		backoff          = initialBackoff
		authFailures     = 0
		consecutiveFails = 0
	)

	for {
		// Immediate context check
		if ctx.Err() != nil {
			log.Println("[EventMonitor] Context canceled, exiting monitor loop")
			return
		}

		// Step 1: Setup and token with circuit breaker
		log.Println("[EventMonitor] Initializing auth...")
		if err := setupAuth(ctx, em.yeastarService); err != nil {
			authFailures++
			if authFailures >= maxAuthRetries {
				log.Printf("[EventMonitor] CRITICAL: Auth setup failed %d times: %v", authFailures, err)
				if !em.waitWithContext(ctx, 1*time.Hour) { // Long cooldown
					return
				}
				authFailures = 0 // Reset after cooldown
				continue
			}

			log.Printf("[EventMonitor] Auth setup failed (attempt %d/%d): %v",
				authFailures, maxAuthRetries, err)
			if !em.waitWithContext(ctx, backoff) {
				return
			}
			backoff = min(backoff*2, maxBackoff)
			continue
		}

		// Step 2: Token acquisition
		log.Println("[EventMonitor] Ensuring valid token...")
		if err := em.yeastarService.EnsureValidToken(ctx); err != nil {
			log.Printf("[EventMonitor] Token error: %v", err)
			if errors.Is(err, ErrInvalidCredentials) {
				log.Fatal("[EventMonitor] FATAL: Invalid credentials")
			}

			if !em.waitWithContext(ctx, backoff) {
				return
			}
			backoff = min(backoff*2, maxBackoff)
			continue
		}

		// Reset failure counters on successful auth
		authFailures = 0
		backoff = initialBackoff

		// Step 3: WebSocket connection
		log.Println("[EventMonitor] Connecting WebSocket...")
		if err := em.webSocketService.Connect(ctx); err != nil {
			log.Printf("[EventMonitor] WebSocket error: %v", err)
			if !em.waitWithContext(ctx, backoff) {
				return
			}
			backoff = min(backoff*2, maxBackoff)
			continue
		}

		// Step 4: Subscription
		log.Println("[EventMonitor] Subscribing to events...")
		if err := em.webSocketService.Subscribe(eventIDs); err != nil {
			log.Printf("[EventMonitor] Subscribe error: %v", err)
			em.webSocketService.Close()
			if !em.waitWithContext(ctx, backoff) {
				return
			}
			backoff = min(backoff*2, maxBackoff)
			continue
		}

		// Step 5: Heartbeat
		em.webSocketService.StartHeartbeat()
		defer func() {
			em.webSocketService.StopHeartbeat()
			log.Println("[EventMonitor] Heartbeat stopped")
		}()

		// Step 6: Event processing
		log.Println("[EventMonitor] Listening for events...")
		err := em.webSocketService.Listen(ctx)
		if err != nil {
			consecutiveFails++
			log.Printf("[EventMonitor] Listen error (%d consecutive): %v", consecutiveFails, err)

			// Critical failure threshold
			if consecutiveFails > 10 {
				log.Println("[EventMonitor] Too many consecutive failures, entering cooldown")
				if !em.waitWithContext(ctx, 5*time.Minute) {
					return
				}
				consecutiveFails = 0
			}
		} else {
			consecutiveFails = 0
		}

		// Cleanup before retry
		em.webSocketService.Close()
		log.Printf("[EventMonitor] Disconnected. Next attempt in %v", backoff)
		if !em.waitWithContext(ctx, backoff) {
			return
		}
		backoff = min(backoff*2, maxBackoff)
	}
}

// Helper method for context-aware waiting
func (em *EventMonitor) waitWithContext(ctx context.Context, duration time.Duration) bool {
	select {
	case <-time.After(duration):
		return true
	case <-ctx.Done():
		log.Println("[EventMonitor] Wait interrupted by context cancellation")
		return false
	}
}

// Helper to get minimum duration
func min(a, b time.Duration) time.Duration {
	if a < b {
		return a
	}
	return b
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
