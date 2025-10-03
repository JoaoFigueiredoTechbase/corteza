package Yeastar

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"runtime"
	"strconv"
	"sync"
	"time"
)

// CortezaClient handles communication with Corteza
type CortezaClient struct {
	client      *http.Client
	baseURL     string
	moduleLocks map[string]*sync.Mutex
	locksMutex  sync.RWMutex
}

func (cc *CortezaClient) getModuleLock(moduleName string) *sync.Mutex {
	cc.locksMutex.RLock()
	lock, exists := cc.moduleLocks[moduleName]
	cc.locksMutex.RUnlock()

	if exists {
		return lock
	}

	cc.locksMutex.Lock()
	defer cc.locksMutex.Unlock()

	if cc.moduleLocks == nil {
		cc.moduleLocks = make(map[string]*sync.Mutex)
	}

	lock = &sync.Mutex{}
	cc.moduleLocks[moduleName] = lock
	return lock
}

// NewCortezaClient creates a new Corteza client
func NewCortezaClient(baseURL string) *CortezaClient {
	return &CortezaClient{
		client: &http.Client{
			Timeout: 60 * time.Minute,
		},
		baseURL: baseURL,
	}
}

func (cc *CortezaClient) GetConfig() (*Config, error) {
	url := fmt.Sprintf("%s/api/gateway/get/config", cc.baseURL)
	fmt.Printf("[CortezaClient] Requesting config: %s\n", url)

	resp, err := cc.client.Get(url)
	if err != nil {
		return nil, fmt.Errorf("failed to get config: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("unexpected status %d: %s", resp.StatusCode, string(body))
	}

	var cfg Config
	if err := json.NewDecoder(resp.Body).Decode(&cfg); err != nil {
		return nil, fmt.Errorf("invalid config JSON: %w", err)
	}

	fmt.Printf("[CortezaClient] Config received: %+v\n", cfg)
	return &cfg, nil
}

func (cc *CortezaClient) GetToken() (*TokenResponse, error) {
	url := fmt.Sprintf("%s/api/gateway/get/token", cc.baseURL)
	fmt.Printf("[CortezaClient] Requesting token: %s\n", url)

	resp, err := cc.client.Get(url)
	if err != nil {
		return nil, fmt.Errorf("failed to get token: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("unexpected status %d: %s", resp.StatusCode, string(body))
	}

	var token TokenResponse
	if err := json.NewDecoder(resp.Body).Decode(&token); err != nil {
		return nil, fmt.Errorf("invalid token JSON: %w", err)
	}

	fmt.Printf("[CortezaClient] Token received: %+v\n", token)
	return &token, nil
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

func getGoroutineID() uint64 {
	b := make([]byte, 64)
	b = b[:runtime.Stack(b, false)]
	b = bytes.TrimPrefix(b, []byte("goroutine "))
	b = b[:bytes.IndexByte(b, ' ')]
	n, _ := strconv.ParseUint(string(b), 10, 64)
	return n
}

func (cc *CortezaClient) SendData(ctx context.Context, moduleName string, data interface{}) error {
	// fmt.Println("Starting SendData to Corteza")

	lock := cc.getModuleLock(moduleName)
	lock.Lock()
	defer lock.Unlock()
	start := time.Now()
	log.Printf("[CORTEZA] SendData START - module: %s, goroutine: %d", moduleName, getGoroutineID())

	defer func() {
		log.Printf("[CORTEZA] SendData END - module: %s, duration: %v", moduleName, time.Since(start))
	}()

	// // Log base URL and module
	// fmt.Printf("Corteza BaseURL: %s\n", cc.baseURL)
	// fmt.Printf("Module Name: %s\n", moduleName)

	// Construct full endpoint
	url := fmt.Sprintf("%s/api/gateway/%s/sync", cc.baseURL, moduleName)
	//fmt.Printf("Full Sync Endpoint: %s\n", url)

	// Construct payload
	payload := map[string]interface{}{
		"data": data,
	}

	// Marshal payload
	jsonPayload, err := json.MarshalIndent(payload, "", "  ") // Pretty-print
	if err != nil {
		//fmt.Printf("Error marshaling payload for module %s: %v\n", moduleName, err)
		return fmt.Errorf("failed to marshal data for module %s: %w", moduleName, err)
	}
	// fmt.Printf("JSON Payload:\n%s\n", string(jsonPayload))

	// Create filename with timestamp
	// timeStamp := time.Now().Format("20060102_150405") // YYYYMMDD_HHMMSS
	// fileName := fmt.Sprintf("payload_%s_%s.txt", moduleName, timeStamp)

	// err = os.WriteFile(fileName, jsonPayload, 0644)
	// if err != nil {
	// 	return fmt.Errorf("failed to write payload to file for module %s: %w", moduleName, err)
	// }
	// fmt.Printf("Payload written to file: %s\n", fileName)

	// Create HTTP request
	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(jsonPayload))
	if err != nil {
		fmt.Printf("Failed to create HTTP request: %v\n", err)
		return fmt.Errorf("failed to create request for module %s: %w", moduleName, err)
	}
	req.Header.Set("Content-Type", "application/json")

	// Log request metadata
	//fmt.Printf("Request Method: %s\n", req.Method)
	//fmt.Printf("Request Headers:\n")
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
	// fmt.Printf("Response Status: %d %s\n", resp.StatusCode, resp.Status)
	// fmt.Printf("Response Headers:\n")
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

	duration := time.Since(start)
	if duration > 5*time.Second {
		log.Printf("WARNING: SendData to %s took %v - possible collision", moduleName, duration)
	}

	return nil
}
