package yeastar

type CallEvent struct {
	Event      string `json:"event"`     // e.g. "incoming", "outgoing"
	CallerID   string `json:"caller_id"` // Number calling
	CalleeID   string `json:"callee_id"` // Number being called
	Timestamp  string `json:"timestamp"` // ISO 8601 timestamp
	Direction  string `json:"direction"` // inbound/outbound
	CallStatus string `json:"status"`    // e.g. ringing, answered, ended
	CallID     string `json:"call_id"`   // Yeastar unique call ID
}

type APIResponse struct {
	Success bool   `json:"success"`
	Error   string `json:"error,omitempty"`
}
