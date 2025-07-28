package yeastarEvents

import (
	"context"
	"log"
	"time"
)

// EventProcessor handles the business logic of processing events
type EventProcessor struct {
	eventChan     chan map[string]interface{}
	cortezaSender *CortezaSender
}

func NewEventProcessor(cortezaSender *CortezaSender) *EventProcessor {
	return &EventProcessor{
		eventChan:     make(chan map[string]interface{}, 100),
		cortezaSender: cortezaSender,
	}
}

func (ep *EventProcessor) Start(ctx context.Context) {
	log.Println("Event processor started")

	for {
		select {
		case event := <-ep.eventChan:
			ep.handleEvent(event)
		case <-ctx.Done():
			log.Println("Event processor stopped")
			return
		}
	}
}

func (ep *EventProcessor) ProcessEvent(event map[string]interface{}) {
	// Non-blocking send to channel
	select {
	case ep.eventChan <- event:
		// Event queued successfully
	default:
		log.Println("Event queue full, dropping event")
	}
}

func (ep *EventProcessor) handleEvent(event map[string]interface{}) {
	log.Printf("Processing event: %v", event)

	// Determine event type
	eventType := ep.getEventType(event)

	switch eventType {
	case "call_status":
		ep.handleCallStatus(event)
	case "call_record":
		ep.handleCallRecord(event)
	case "extension_status":
		ep.handleExtensionStatus(event)
	default:
		log.Printf("Unknown event type: %s", eventType)
	}
}

func (ep *EventProcessor) getEventType(event map[string]interface{}) string {
	// Try to determine event type from the event data
	if eventType, ok := event["event_type"].(string); ok {
		return eventType
	}

	// Try numeric type (like 30011, 30012)
	if typeNum, ok := event["type"].(float64); ok {
		switch int(typeNum) {
		case 30011:
			return "call_status"
		case 30012:
			return "call_record"
		case 30014:
			return "extension_status"
		}
	}

	return "unknown"
}

func (ep *EventProcessor) handleCallStatus(event map[string]interface{}) {
	log.Println("Handling call status event")

	// Create a simple call event for Corteza
	callEvent := CortezaCallEvent{
		Type:      "call_status_update",
		CallID:    ep.getString(event, "call_id"),
		CallerID:  ep.getString(event, "caller_id"),
		CalleeID:  ep.getString(event, "callee_id"),
		Status:    ep.getString(event, "status"),
		Timestamp: time.Now(),
	}

	// Send to Corteza
	if err := ep.cortezaSender.SendCallEvent(callEvent); err != nil {
		log.Printf("Failed to send call event to Corteza: %v", err)
	}
}

func (ep *EventProcessor) handleCallRecord(event map[string]interface{}) {
	log.Println("Handling call record event")

	// Create a call record for Corteza
	recordEvent := CortezaCallRecord{
		Type:      "call_completed",
		CallID:    ep.getString(event, "call_id"),
		Duration:  ep.getInt(event, "duration"),
		Cost:      ep.getFloat(event, "cost"),
		Timestamp: time.Now(),
	}

	if err := ep.cortezaSender.SendCallRecord(recordEvent); err != nil {
		log.Printf("Failed to send call record to Corteza: %v", err)
	}
}

func (ep *EventProcessor) handleExtensionStatus(event map[string]interface{}) {
	log.Println("Handling extension status event")
	// Implement extension status handling
}

// Helper functions to safely extract values
func (ep *EventProcessor) getString(event map[string]interface{}, key string) string {
	if val, ok := event[key].(string); ok {
		return val
	}
	return ""
}

func (ep *EventProcessor) getInt(event map[string]interface{}, key string) int {
	if val, ok := event[key].(float64); ok {
		return int(val)
	}
	return 0
}

func (ep *EventProcessor) getFloat(event map[string]interface{}, key string) float64 {
	if val, ok := event[key].(float64); ok {
		return val
	}
	return 0.0
}
