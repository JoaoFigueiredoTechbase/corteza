package Yeastar

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

// CortezaClient handles communication with Corteza
type CortezaClient struct {
	client  *http.Client
	baseURL string
}

// NewCortezaClient creates a new Corteza client
func NewCortezaClient(baseURL string) *CortezaClient {
	return &CortezaClient{
		client: &http.Client{
			Timeout: 30 * time.Second,
		},
		baseURL: baseURL,
	}
}

// TriggerConfigPush triggers configuration push from Corteza
func (cc *CortezaClient) TriggerConfigPush() error {
	url := fmt.Sprintf("%s/api/gateway/get/config", cc.baseURL)

	resp, err := cc.client.Get(url)
	if err != nil {
		return fmt.Errorf("failed to trigger config push: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("Corteza config trigger failed: %d, body: %s", resp.StatusCode, string(body))
	}

	return nil
}

// TriggerTokenPush triggers token push from Corteza
func (cc *CortezaClient) TriggerTokenPush() error {
	url := fmt.Sprintf("%s/api/gateway/get/token", cc.baseURL)

	resp, err := cc.client.Get(url)
	if err != nil {
		return fmt.Errorf("failed to trigger token push: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("Corteza token trigger failed: %d, body: %s", resp.StatusCode, string(body))
	}

	return nil
}

// SaveToken saves token to Corteza storage
func (cc *CortezaClient) SaveToken(ctx context.Context, token *TokenResponse) error {
	url := fmt.Sprintf("%s/api/gateway/store/token", cc.baseURL)

	fmt.Printf("Saving Token to Corteza: %+v\n", token)

	jsonPayload, err := json.Marshal(token)
	if err != nil {
		return fmt.Errorf("failed to marshal token for saving: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(jsonPayload))
	if err != nil {
		return fmt.Errorf("failed to create save request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := cc.client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send token to storage API: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("failed to store token remotely, status: %d, body: %s", resp.StatusCode, string(body))
	}

	return nil
}

// SendData sends processed data to Corteza using specific sync endpoints.
func (cc *CortezaClient) SendData(ctx context.Context, moduleName string, data interface{}) error {
	// Convert plural moduleName to singular for the endpoint path
	// Example: "agents" -> "agent", "queues" -> "queue", "cdrs" -> "cdr"
	singularModuleName := strings.TrimSuffix(moduleName, "s") // Simple plural to singular for these cases

	// Construct the URL using the specific sync endpoint
	url := fmt.Sprintf("%s/api/gateway/%s/sync/", cc.baseURL, singularModuleName)

	jsonPayload, err := json.Marshal(data)
	if err != nil {
		return fmt.Errorf("failed to marshal data for saving to Corteza module %s: %w", moduleName, err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(jsonPayload))
	if err != nil {
		return fmt.Errorf("failed to create request for module %s: %w", moduleName, err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := cc.client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send data to Corteza storage API for module %s: %w", moduleName, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body) // Consider logging readErr here too
		return fmt.Errorf("failed to store data remotely for module %s, status: %d, response: %s",
			moduleName, resp.StatusCode, string(bodyBytes))
	}

	return nil
}
