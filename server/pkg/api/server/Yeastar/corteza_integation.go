package Yeastar

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
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
	fmt.Printf("[CortezaClient] Triggering config push: %s\n", url)

	resp, err := cc.client.Get(url)
	if err != nil {
		fmt.Printf("[CortezaClient]   Config trigger request failed: %v\n", err)
		return fmt.Errorf("failed to trigger config push: %w", err)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	fmt.Printf("[CortezaClient] Config trigger response: %d - %s\n", resp.StatusCode, string(body))

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("Corteza config trigger failed: %d, body: %s", resp.StatusCode, string(body))
	}

	fmt.Println("[CortezaClient]   Config push triggered successfully")
	return nil
}

// TriggerTokenPush triggers token push from Corteza
func (cc *CortezaClient) TriggerTokenPush() error {
	url := fmt.Sprintf("%s/api/gateway/get/token", cc.baseURL)
	fmt.Printf("[CortezaClient] Triggering token push: %s\n", url)

	resp, err := cc.client.Get(url)
	if err != nil {
		fmt.Printf("[CortezaClient]   Token trigger request failed: %v\n", err)
		return fmt.Errorf("failed to trigger token push: %w", err)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	fmt.Printf("[CortezaClient] Token trigger response: %d - %s\n", resp.StatusCode, string(body))

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("Corteza token trigger failed: %d, body: %s", resp.StatusCode, string(body))
	}

	fmt.Println("[CortezaClient]   Token push triggered successfully")
	return nil
}

// SaveToken saves token to Corteza storage
func (cc *CortezaClient) SaveToken(ctx context.Context, token *TokenResponse) error {
	fmt.Println("[CortezaClient] Starting to save token to Corteza...")

	url := fmt.Sprintf("%s/api/gateway/store/token", cc.baseURL)
	fmt.Printf("[CortezaClient] Save token URL: %s\n", url)

	jsonPayload, err := json.Marshal(token)
	if err != nil {
		fmt.Printf("[CortezaClient]   Failed to marshal token: %v\n", err)
		return fmt.Errorf("failed to marshal token for saving: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(jsonPayload))
	if err != nil {
		fmt.Printf("[CortezaClient]   Failed to create request: %v\n", err)
		return fmt.Errorf("failed to create save request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	fmt.Println("[CortezaClient] Sending token to Corteza storage API...")
	resp, err := cc.client.Do(req)
	if err != nil {
		fmt.Printf("[CortezaClient] Failed to send request: %v\n", err)
		return fmt.Errorf("failed to send token to storage API: %w", err)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	fmt.Printf("[CortezaClient] Save token response: %d - %s\n", resp.StatusCode, string(body))

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("failed to store token remotely, status: %d, body: %s", resp.StatusCode, string(body))
	}

	fmt.Println("[CortezaClient] Token successfully saved to Corteza")
	return nil
}

// SendData sends processed data to Corteza using specific sync endpoints.
func (cc *CortezaClient) SendData(ctx context.Context, moduleName string, data interface{}) error {
	fmt.Println("Starting SendData to Corteza")

	// Log base URL and module
	fmt.Printf("Corteza BaseURL: %s\n", cc.baseURL)
	fmt.Printf("Module Name: %s\n", moduleName)

	// Construct full endpoint
	url := fmt.Sprintf("%s/api/gateway/%s/sync", cc.baseURL, moduleName)
	fmt.Printf("Full Sync Endpoint: %s\n", url)

	// Construct payload
	payload := map[string]interface{}{
		"data": data,
	}

	// Marshal payload
	jsonPayload, err := json.MarshalIndent(payload, "", "  ") // Pretty-print
	if err != nil {
		fmt.Printf("Error marshaling payload for module %s: %v\n", moduleName, err)
		return fmt.Errorf("failed to marshal data for module %s: %w", moduleName, err)
	}
	// fmt.Printf("JSON Payload:\n%s\n", string(jsonPayload))

	// Create HTTP request
	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(jsonPayload))
	if err != nil {
		fmt.Printf("Failed to create HTTP request: %v\n", err)
		return fmt.Errorf("failed to create request for module %s: %w", moduleName, err)
	}
	req.Header.Set("Content-Type", "application/json")

	// Log request metadata
	fmt.Printf("Request Method: %s\n", req.Method)
	fmt.Printf("Request Headers:\n")
	for k, v := range req.Header {
		fmt.Printf("  %s: %v\n", k, v)
	}

	// Send request
	fmt.Println("Sending request to Corteza...")
	resp, err := cc.client.Do(req)
	if err != nil {
		fmt.Printf("Request error for module %s: %v\n", moduleName, err)
		return fmt.Errorf("failed to send data to Corteza for module %s: %w", moduleName, err)
	}
	defer resp.Body.Close()

	// Read response body
	bodyBytes, _ := io.ReadAll(resp.Body)

	// Log response details
	fmt.Printf("Response Status: %d %s\n", resp.StatusCode, resp.Status)
	fmt.Printf("Response Headers:\n")
	for k, v := range resp.Header {
		fmt.Printf("  %s: %v\n", k, v)
	}
	fmt.Printf("Response Body:\n%s\n", string(bodyBytes))

	if resp.StatusCode != http.StatusOK {
		fmt.Printf("Unexpected status code: %d\n", resp.StatusCode)
		return fmt.Errorf("failed to store data for module %s, status: %d, response: %s",
			moduleName, resp.StatusCode, string(bodyBytes))
	}

	fmt.Printf("Data for module '%s' stored successfully!\n", moduleName)
	return nil
}
