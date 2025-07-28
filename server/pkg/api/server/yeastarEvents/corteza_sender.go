package yeastarEvents

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"
)

// CortezaSender handles sending events to Corteza
type CortezaSender struct {
	baseURL    string
	httpClient *http.Client
}

func NewCortezaSender(baseURL string) *CortezaSender {
	return &CortezaSender{
		baseURL: baseURL,
		httpClient: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

func (cs *CortezaSender) SendCallEvent(event CortezaCallEvent) error {
	log.Printf("Sending call event to Corteza: %+v", event)

	// Convert to JSON
	jsonData, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("failed to marshal event: %w", err)
	}

	// Send to Corteza
	url := fmt.Sprintf("%s/api/calls/events", cs.baseURL)
	resp, err := cs.httpClient.Post(url, "application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		return fmt.Errorf("Corteza returned error status: %d", resp.StatusCode)
	}

	log.Println("Call event sent successfully to Corteza")
	return nil
}

func (cs *CortezaSender) SendCallRecord(record CortezaCallRecord) error {
	log.Printf("Sending call record to Corteza: %+v", record)

	jsonData, err := json.Marshal(record)
	if err != nil {
		return fmt.Errorf("failed to marshal record: %w", err)
	}

	url := fmt.Sprintf("%s/api/calls/records", cs.baseURL)
	resp, err := cs.httpClient.Post(url, "application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		return fmt.Errorf("Corteza returned error status: %d", resp.StatusCode)
	}

	log.Println("Call record sent successfully to Corteza")
	return nil
}
