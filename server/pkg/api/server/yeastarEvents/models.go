package yeastarEvents

import "time"

// CortezaCallEvent represents a call event we send to Corteza
type CortezaCallEvent struct {
	Type      string    `json:"type"`      // "call_status_update"
	CallID    string    `json:"call_id"`   // Unique call identifier
	CallerID  string    `json:"caller_id"` // Who's calling
	CalleeID  string    `json:"callee_id"` // Who's being called
	Status    string    `json:"status"`    // "ringing", "answered", "ended"
	Timestamp time.Time `json:"timestamp"` // When this happened
}

// CortezaCallRecord represents a completed call record
type CortezaCallRecord struct {
	Type      string    `json:"type"`      // "call_completed"
	CallID    string    `json:"call_id"`   // Unique call identifier
	Duration  int       `json:"duration"`  // Call duration in seconds
	Cost      float64   `json:"cost"`      // Call cost
	Timestamp time.Time `json:"timestamp"` // When call ended
}

// YeastarEvent represents the raw event from Yeastar
type YeastarEvent struct {
	Type int                    `json:"type"` // Event type number (30011, 30012, etc.)
	Data map[string]interface{} `json:"data"` // Event data
}
