// events.go
package yeastar

import (
	"context"
	"fmt"
	"sync"
	"time"

	"go.uber.org/zap"
)

// EventProcessor handles incoming Yeastar events
type EventProcessor struct {
	log       *zap.Logger
	client    *Client
	eventChan chan map[string]interface{}
	handlers  map[int]func(map[string]interface{}) error
	wg        sync.WaitGroup
}

func NewEventProcessor(log *zap.Logger, client *Client) *EventProcessor {
	return &EventProcessor{
		log:       log.Named("yeastar.processor"),
		client:    client,
		eventChan: make(chan map[string]interface{}, 100),
		handlers:  make(map[int]func(map[string]interface{}) error),
	}
}

// Start begins processing events
func (ep *EventProcessor) Start(ctx context.Context) {
	ep.wg.Add(1)
	go func() {
		defer ep.wg.Done()
		for {
			select {
			case event := <-ep.eventChan:
				ep.processEvent(event)
			case <-ctx.Done():
				return
			}
		}
	}()
}

// AddHandler registers an event handler
func (ep *EventProcessor) AddHandler(eventType int, handler func(map[string]interface{}) error) {
	ep.handlers[eventType] = handler
}

// Enqueue adds an event to the processing queue
func (ep *EventProcessor) Enqueue(event map[string]interface{}) {
	select {
	case ep.eventChan <- event:
		ep.log.Debug("Event enqueued")
	default:
		ep.log.Warn("Event queue full, dropping event")
	}
}

// processEvent routes events to appropriate handlers
func (ep *EventProcessor) processEvent(event map[string]interface{}) {
	start := time.Now()
	eventType, ok := event["type"].(float64)
	if !ok {
		ep.log.Error("Event missing type field")
		return
	}
	typeInt := int(eventType)

	if handler, exists := ep.handlers[typeInt]; exists {
		if err := handler(event); err != nil {
			ep.log.Error("Error processing event",
				zap.Int("type", typeInt),
				zap.Error(err))
		}
	} else {
		ep.log.Debug("Unhandled event type", zap.Int("type", typeInt))
	}

	ep.log.Debug("Processed event",
		zap.Int("type", typeInt),
		zap.Duration("duration", time.Since(start)),
	)
}

// BroadcastEvent sends event to Corteza
func (ep *EventProcessor) BroadcastEvent(ctx context.Context, eventType string, data map[string]interface{}) error {
	event := map[string]interface{}{
		"event": eventType,
		"data":  data,
		"ts":    time.Now().Unix(),
	}

	return ep.client.SendCallNotification(ctx, CallEvent{
		Event:      eventType,
		CallerID:   fmt.Sprint(data["caller_id"]),
		CalleeID:   fmt.Sprint(data["callee_id"]),
		Timestamp:  time.Now().Format(time.RFC3339),
		Direction:  fmt.Sprint(data["direction"]),
		CallStatus: fmt.Sprint(data["status"]),
		CallID:     fmt.Sprint(data["call_id"]),
	})
}
