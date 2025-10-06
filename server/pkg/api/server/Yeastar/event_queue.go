package Yeastar

import (
	"context"
	"fmt"
	"log"
	"sync"
	"sync/atomic"
	"time"
)

// EventQueue manages concurrent event processing with a worker pool
type EventQueue struct {
	eventChan     chan map[string]interface{}
	workerCount   int
	stopChan      chan struct{}
	wg            sync.WaitGroup
	mu            sync.RWMutex
	isRunning     bool
	cortezaClient *CortezaClient

	// Metrics
	processedCount int64
	errorCount     int64
	droppedCount   int64
}

// EventQueueConfig holds configuration for the event queue
type EventQueueConfig struct {
	BufferSize  int // Size of the event buffer
	WorkerCount int // Number of concurrent workers
}

// DefaultEventQueueConfig returns sensible defaults
func DefaultEventQueueConfig() EventQueueConfig {
	return EventQueueConfig{
		BufferSize:  1000, // Can hold 1000 events before blocking
		WorkerCount: 10,   // 10 concurrent workers
	}
}

// NewEventQueue creates a new event queue with the given configuration
func NewEventQueue(config EventQueueConfig, cortezaClient *CortezaClient) *EventQueue {
	if config.BufferSize <= 0 {
		config.BufferSize = 1000
	}
	if config.WorkerCount <= 0 {
		config.WorkerCount = 10
	}

	return &EventQueue{
		eventChan:     make(chan map[string]interface{}, config.BufferSize),
		workerCount:   config.WorkerCount,
		stopChan:      make(chan struct{}),
		cortezaClient: cortezaClient,
	}
}

// Start begins processing events with the worker pool
func (eq *EventQueue) Start(ctx context.Context) error {
	eq.mu.Lock()
	defer eq.mu.Unlock()

	if eq.isRunning {
		return fmt.Errorf("event queue already running")
	}

	eq.isRunning = true
	log.Printf("[EventQueue] Starting with %d workers and buffer size %d", eq.workerCount, cap(eq.eventChan))

	// Start worker goroutines
	for i := 0; i < eq.workerCount; i++ {
		eq.wg.Add(1)
		go eq.worker(ctx, i)
	}

	// Start metrics logger
	go eq.logMetrics()

	return nil
}

// worker processes events from the queue
func (eq *EventQueue) worker(ctx context.Context, workerID int) {
	defer eq.wg.Done()

	log.Printf("[EventQueue] Worker %d started", workerID)

	for {
		select {
		case <-ctx.Done():
			log.Printf("[EventQueue] Worker %d stopping due to context cancellation", workerID)
			return
		case <-eq.stopChan:
			log.Printf("[EventQueue] Worker %d stopping due to stop signal", workerID)
			return
		case event, ok := <-eq.eventChan:
			if !ok {
				log.Printf("[EventQueue] Worker %d stopping - channel closed", workerID)
				return
			}

			// Process the event with timeout
			if err := eq.processEventWithTimeout(ctx, event, 30*time.Second); err != nil {
				atomic.AddInt64(&eq.errorCount, 1)
				log.Printf("[EventQueue] Worker %d error processing event: %v", workerID, err)
			} else {
				atomic.AddInt64(&eq.processedCount, 1)
			}
		}
	}
}

// processEventWithTimeout processes a single event with a timeout
func (eq *EventQueue) processEventWithTimeout(ctx context.Context, event map[string]interface{}, timeout time.Duration) error {
	// Create a context with timeout for this specific event
	eventCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	// Channel to receive the result
	done := make(chan error, 1)

	// Process event in a goroutine
	go func() {
		done <- eq.processEvent(eventCtx, event)
	}()

	// Wait for completion or timeout
	select {
	case err := <-done:
		return err
	case <-eventCtx.Done():
		return fmt.Errorf("event processing timeout after %v", timeout)
	}
}

// processEvent handles the actual event processing logic
func (eq *EventQueue) processEvent(ctx context.Context, event map[string]interface{}) error {
	eventID, _ := event["type"].(float64)
	sn, _ := event["sn"].(string)

	if eventID == 0 {
		// Likely a heartbeat acknowledgment, skip
		return nil
	}

	log.Printf("[EventQueue] Processing event - SN: %s, ID: %.0f", sn, eventID)

	var err error
	switch int(eventID) {
	case EventExtensionRegistration:
		_, err = handleEventExtensionRegistration(event)
	case EventExtensionCallStatus:
		_, err = handleEventExtensionCallStatus(event)
	case EventExtensionPresenceStatus:
		_, err = handleEventExtensionPresenceStatus(event)
	case EventCallStatusChanged:
		_, err = handleEventCallStatusChanged(event)
	case EventNewCDR:
		_, err = handleEventNewCDR(event)
	case EventCallTransfer:
		_, err = handleEventCallTransfer(event)
	case EventCallFoward:
		_, err = handleEventCallFoward(event)
	case EventCallFailed:
		_, err = handleEventCallFailedStatus(event)
	case EventSatisfaction:
		_, err = handleEventSatisfaction(event)
	case EventExtensionConfiguration:
		_, err = handleEventExtensionConfiguration(event)
	case EventAgentPause:
		_, err = handleEventAgentPause(event)
	case EventAgentRingTimeout:
		_, err = handleEventAgentRingTimeout(event)
	case EventCallNoteStatusChanged:
		_, err = handleEventCallNoteStatusChanged(event)
	case EventAgentStatusChanged:
		_, err = handleEventAgentStatusChanged(event)
	default:
		log.Printf("[EventQueue] Unknown event ID: %.0f", eventID)
		return fmt.Errorf("unknown event ID: %.0f", eventID)
	}

	if err != nil {
		return fmt.Errorf("handler error for event %.0f: %w", eventID, err)
	}

	return nil
}

// Enqueue adds an event to the processing queue
// Returns false if the queue is full (non-blocking)
func (eq *EventQueue) Enqueue(event map[string]interface{}) bool {
	select {
	case eq.eventChan <- event:
		return true
	default:
		// Queue is full, drop the event
		atomic.AddInt64(&eq.droppedCount, 1)
		log.Printf("[EventQueue] WARNING: Queue full, event dropped. Consider increasing buffer size or worker count.")
		return false
	}
}

// EnqueueBlocking adds an event to the queue, blocking if full
func (eq *EventQueue) EnqueueBlocking(event map[string]interface{}) {
	eq.eventChan <- event
}

// Stop gracefully shuts down the event queue
func (eq *EventQueue) Stop() {
	eq.mu.Lock()
	defer eq.mu.Unlock()

	if !eq.isRunning {
		return
	}

	log.Println("[EventQueue] Stopping event queue...")
	close(eq.stopChan)

	// Close the event channel to signal workers
	close(eq.eventChan)

	// Wait for all workers to finish
	eq.wg.Wait()

	eq.isRunning = false
	log.Println("[EventQueue] Event queue stopped")

	// Log final metrics
	eq.printMetrics()
}

// GetMetrics returns current queue metrics
func (eq *EventQueue) GetMetrics() (processed, errors, dropped int64, queueDepth int) {
	return atomic.LoadInt64(&eq.processedCount),
		atomic.LoadInt64(&eq.errorCount),
		atomic.LoadInt64(&eq.droppedCount),
		len(eq.eventChan)
}

// printMetrics logs current metrics
func (eq *EventQueue) printMetrics() {
	processed, errors, dropped, depth := eq.GetMetrics()
	log.Printf("[EventQueue] Metrics - Processed: %d, Errors: %d, Dropped: %d, Queue Depth: %d",
		processed, errors, dropped, depth)
}

// logMetrics periodically logs queue metrics
func (eq *EventQueue) logMetrics() {
	ticker := time.NewTicker(60 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-eq.stopChan:
			return
		case <-ticker.C:
			eq.printMetrics()
		}
	}
}

// IsRunning returns whether the queue is currently running
func (eq *EventQueue) IsRunning() bool {
	eq.mu.RLock()
	defer eq.mu.RUnlock()
	return eq.isRunning
}

// QueueDepth returns the current number of events waiting in the queue
func (eq *EventQueue) QueueDepth() int {
	return len(eq.eventChan)
}
