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
	fmt.Println("💾 Starting to save token to Corteza")

	url := fmt.Sprintf("%s/api/gateway/store/token", cc.baseURL)
	fmt.Printf("🌍 Save token URL: %s\n", url)

	fmt.Printf("🔐 Token being saved: %+v\n", token)

	jsonPayload, err := json.Marshal(token)
	if err != nil {
		fmt.Printf("❌ Failed to marshal token: %v\n", err)
		return fmt.Errorf("failed to marshal token for saving: %w", err)
	}
	fmt.Printf("📤 JSON payload: %s\n", string(jsonPayload))

	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(jsonPayload))
	if err != nil {
		fmt.Printf("❌ Failed to create request: %v\n", err)
		return fmt.Errorf("failed to create save request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	fmt.Println("📡 Sending token to Corteza token storage API")
	resp, err := cc.client.Do(req)
	if err != nil {
		fmt.Printf("❌ Failed to send request: %v\n", err)
		return fmt.Errorf("failed to send token to storage API: %w", err)
	}
	defer resp.Body.Close()

	fmt.Printf("📥 Received response with status code: %d\n", resp.StatusCode)

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		fmt.Printf("❌ Token storage failed: %d - %s\n", resp.StatusCode, string(body))
		return fmt.Errorf("failed to store token remotely, status: %d, body: %s", resp.StatusCode, string(body))
	}

	fmt.Println("✅ Token successfully saved to Corteza")
	return nil
}

// SendData sends processed data to Corteza using specific sync endpoints.
func (cc *CortezaClient) SendData(ctx context.Context, moduleName string, data interface{}) error {
	// Construct the URL using the specific sync endpoint
	url := fmt.Sprintf("%s/api/gateway/%s/sync/", cc.baseURL, moduleName)

	fmt.Printf("📡 Sending data to Corteza\n")
	fmt.Printf("🔗 Endpoint: %s\n", url)

	payload := map[string]interface{}{
		"data": data,
	}

	jsonPayload, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("❌ failed to marshal data for module %s: %w", moduleName, err)
	}

	fmt.Printf("📦 Payload for module %s:\n%s\n", moduleName, string(jsonPayload))

	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(jsonPayload))
	if err != nil {
		return fmt.Errorf("❌ failed to create request for module %s: %w", moduleName, err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := cc.client.Do(req)
	if err != nil {
		return fmt.Errorf("❌ failed to send data to Corteza for module %s: %w", moduleName, err)
	}
	defer resp.Body.Close()

	bodyBytes, _ := io.ReadAll(resp.Body)

	fmt.Printf("📥 Response from Corteza for module %s:\n", moduleName)
	fmt.Printf("🔢 Status code: %d\n", resp.StatusCode)
	fmt.Printf("📄 Response body: %s\n", string(bodyBytes))

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("❌ failed to store data for module %s, status: %d, response: %s",
			moduleName, resp.StatusCode, string(bodyBytes))
	}

	fmt.Printf("✅ Data for module %s stored successfully!\n", moduleName)
	return nil
}
