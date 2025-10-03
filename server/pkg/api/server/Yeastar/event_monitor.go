package Yeastar

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net"
	"sync/atomic"
	"time"
)

type EventMonitor struct {
	yeastarService   *YeastarService
	webSocketService *WebSocketService
	isRunning        bool
	syncInProgress   atomic.Bool
	eventQueue       *EventQueue
}

var (
	ErrInvalidCredentials = errors.New("invalid credentials")
	ErrNotConnected       = errors.New("websocket not connected")
	ErrSubscriptionFailed = errors.New("subscription failed")
)

func NewEventMonitor(configManager *ConfigManager, tokenManager *TokenManager, cortezaClient *CortezaClient) *EventMonitor {
	log.Println("[EventMonitor] Initializing Yeastar and WebSocket services")

	yeastarService := NewYeastarService(configManager, tokenManager, cortezaClient)
	
	// Initialize event queue with configuration
	queueConfig := DefaultEventQueueConfig()
	eventQueue := NewEventQueue(queueConfig, cortezaClient)
	
	// Create websocket service and pass the event queue to it
	webSocketService := NewWebSocketService(configManager, tokenManager, cortezaClient, eventQueue)

	webSocketService.SetOnConnect(func() {
		if err := cortezaClient.OnSocketConnect(); err != nil {
			log.Printf("[EventMonitor] OnSocketConnect error: %v", err)
		}
	})

	webSocketService.SetOnDisconnect(func(reason string) {
		if err := cortezaClient.OnSocketDisconnect(); err != nil {
			log.Printf("[EventMonitor] OnSocketDisconnect error: %v", err)
		}
	})

	return &EventMonitor{
		yeastarService:   yeastarService,
		webSocketService: webSocketService,
		eventQueue:       eventQueue,
	}
}

func (em *EventMonitor) Start(ctx context.Context) error {
	if em.isRunning {
		log.Println("[EventMonitor] Start requested but monitor is already running")
		return fmt.Errorf("event monitor is already running")
	}

	log.Println("[EventMonitor] Starting Yeastar event monitor (with automatic retries)...")
	em.isRunning = true

	// Start event queue first
	if err := em.eventQueue.Start(ctx); err != nil {
		log.Printf("[EventMonitor] Failed to start event queue: %v", err)
		em.isRunning = false
		return err
	}

	go em.runMonitorLoop(ctx)

	return nil
}

func (em *EventMonitor) checkConnectivity() bool {
	_, err := net.DialTimeout("tcp", "8.8.8.8:53", 3*time.Second)
	return err == nil
}

func (em *EventMonitor) runMonitorLoop(ctx context.Context) {
	defer func() {
		em.isRunning = false
		em.webSocketService.Close()
		em.eventQueue.Stop()
		log.Println("[EventMonitor] Monitor loop stopped, WebSocket and queue closed")
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
		initialBackoff           = 5 * time.Second
		maxBackoff               = 5 * time.Minute
		maxAuthRetries           = 3
		criticalCooldown         = 20 * time.Minute
		maxConsecutiveFails      = 10
		consecutiveFailsCooldown = 5 * time.Minute
	)

	var (
		backoff          = initialBackoff
		authFailures     = 0
		consecutiveFails = 0
	)

	for {
		if ctx.Err() != nil {
			log.Println("[EventMonitor] Context canceled, exiting monitor loop")
			return
		}

		// Check connectivity first
		if !em.checkConnectivity() {
			log.Print("[EventMonitor] No internet connectivity")
			time.Sleep(30 * time.Second)
			continue
		}

		// Auth setup phase
		log.Println("[EventMonitor] Initializing auth...")
		if err := setupAuth(ctx, em.yeastarService); err != nil {
			authFailures++
			log.Printf("[EventMonitor] Auth setup failed (attempt %d/%d): %v",
				authFailures, maxAuthRetries, err)

			if authFailures >= maxAuthRetries {
				log.Printf("[EventMonitor] CRITICAL: Auth setup failed %d times, entering %v cooldown",
					maxAuthRetries, criticalCooldown)

				if !em.waitWithContext(ctx, criticalCooldown) {
					return
				}

				authFailures = 0
				backoff = initialBackoff
				log.Println("[EventMonitor] Cooldown complete, retrying auth setup...")
				continue
			}

			if !em.waitWithContext(ctx, backoff) {
				return
			}
			backoff = min(backoff*2, maxBackoff)
			continue
		}

		// Token validation phase
		log.Println("[EventMonitor] Ensuring valid token...")
		if err := em.yeastarService.EnsureValidToken(ctx); err != nil {
			log.Printf("[EventMonitor] Token error: %v", err)

			if errors.Is(err, ErrInvalidCredentials) {
				log.Printf("[EventMonitor] CRITICAL: Invalid credentials - cannot continue")
				return
			}

			authFailures++
			if authFailures >= maxAuthRetries {
				log.Printf("[EventMonitor] CRITICAL: Token validation failed %d times, entering %v cooldown",
					maxAuthRetries, criticalCooldown)

				if !em.waitWithContext(ctx, criticalCooldown) {
					return
				}
				authFailures = 0
				backoff = initialBackoff
				continue
			}

			if !em.waitWithContext(ctx, backoff) {
				return
			}
			backoff = min(backoff*2, maxBackoff)
			continue
		}

		// Reset counters after successful auth and token validation
		authFailures = 0
		backoff = initialBackoff

		// WebSocket connection phase
		log.Println("[EventMonitor] Connecting WebSocket...")
		if err := em.webSocketService.Connect(ctx); err != nil {
			consecutiveFails++
			log.Printf("[EventMonitor] Connect failed (%d consecutive): %v (retry in %v)",
				consecutiveFails, err, backoff)

			if consecutiveFails > maxConsecutiveFails {
				log.Printf("[EventMonitor] Too many consecutive connection failures (%d), entering cooldown",
					consecutiveFails)
				if !em.waitWithContext(ctx, consecutiveFailsCooldown) {
					return
				}
				consecutiveFails = 0
				backoff = initialBackoff
				continue
			}

			if !em.waitWithContext(ctx, backoff) {
				return
			}
			backoff = min(backoff*2, maxBackoff)
			continue
		}

		// Subscription phase
		log.Println("[EventMonitor] Subscribing to events...")
		if err := em.webSocketService.Subscribe(eventIDs); err != nil {
			consecutiveFails++
			log.Printf("[EventMonitor] Subscribe error (%d consecutive): %v", consecutiveFails, err)
			em.webSocketService.Close()

			if consecutiveFails > maxConsecutiveFails {
				log.Printf("[EventMonitor] Too many consecutive subscribe failures (%d), entering cooldown",
					consecutiveFails)
				if !em.waitWithContext(ctx, consecutiveFailsCooldown) {
					return
				}
				consecutiveFails = 0
				backoff = initialBackoff
				continue
			}

			if !em.waitWithContext(ctx, backoff) {
				return
			}
			backoff = min(backoff*2, maxBackoff)
			continue
		}

		// Start heartbeat and listen for events
		em.webSocketService.StartHeartbeat()
		log.Println("[EventMonitor] Listening for events...")

		err := em.webSocketService.Listen(ctx)

		// Always stop heartbeat after listening ends
		em.webSocketService.StopHeartbeat()
		em.webSocketService.Close()

		if err != nil {
			consecutiveFails++
			log.Printf("[EventMonitor] Listen error (%d consecutive): %v", consecutiveFails, err)

			if consecutiveFails > maxConsecutiveFails {
				log.Printf("[EventMonitor] Too many consecutive listen failures (%d), entering cooldown",
					consecutiveFails)
				if !em.waitWithContext(ctx, consecutiveFailsCooldown) {
					return
				}
				consecutiveFails = 0
				backoff = initialBackoff
				continue
			}
		} else {
			consecutiveFails = 0
			backoff = initialBackoff
		}

		log.Printf("[EventMonitor] Disconnected. Next attempt in %v", backoff)
		if !em.waitWithContext(ctx, backoff) {
			return
		}
		backoff = min(backoff*2, maxBackoff)
	}
}

func (em *EventMonitor) waitWithContext(ctx context.Context, duration time.Duration) bool {
	select {
	case <-time.After(duration):
		return true
	case <-ctx.Done():
		log.Println("[EventMonitor] Wait interrupted by context cancellation")
		return false
	}
}

func min(a, b time.Duration) time.Duration {
	if a < b {
		return a
	}
	return b
}

func (em *EventMonitor) Stop() error {
	if !em.isRunning {
		log.Println("[EventMonitor] Stop requested but monitor is not running")
		return nil
	}

	log.Println("[EventMonitor] Stopping Yeastar event monitor...")

	if err := em.webSocketService.Close(); err != nil {
		log.Printf("[EventMonitor] Error closing WebSocket: %v", err)
	}

	em.eventQueue.Stop()

	em.isRunning = false
	log.Println("[EventMonitor] Yeastar event monitor stopped successfully")
	return nil
}

func (em *EventMonitor) IsRunning() bool {
	return em.isRunning
}

func (em *EventMonitor) GetQueueMetrics() map[string]interface{} {
	if em.eventQueue == nil {
		return map[string]interface{}{
			"error": "event queue not initialized",
		}
	}

	processed, errors, dropped, depth := em.eventQueue.GetMetrics()
	
	return map[string]interface{}{
		"processed":   processed,
		"errors":      errors,
		"dropped":     dropped,
		"queue_depth": depth,
		"running":     em.eventQueue.IsRunning(),
	}
}

func (em *EventMonitor) IsSyncInProgress() bool {
	return em.syncInProgress.Load()
}

func (em *EventMonitor) SetSyncInProgress(inProgress bool) {
	em.syncInProgress.Store(inProgress)
}