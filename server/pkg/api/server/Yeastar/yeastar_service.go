package Yeastar

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"time"
)

// YeastarService orchestrates API calls with proper token management
type YeastarService struct {
	configManager  *ConfigManager
	tokenManager   *TokenManager
	cortezaClient  *CortezaClient
	httpClient     *http.Client
	baseRetryDelay time.Duration
	maxRetries     int
}

// NewYeastarService creates a new service instance
func NewYeastarService(configManager *ConfigManager, tokenManager *TokenManager, cortezaClient *CortezaClient) *YeastarService {
	return &YeastarService{
		configManager: configManager,
		tokenManager:  tokenManager,
		cortezaClient: cortezaClient,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
			Transport: &http.Transport{
				TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
			},
		},
		baseRetryDelay: 1 * time.Second,
		maxRetries:     3,
	}
}

// WaitForInitialization waits for both config and token to be ready
// func (ys *YeastarService) WaitForInitialization(ctx context.Context) error {
// 	// Wait for config
// 	_, err := ys.configManager.WaitForConfig(ctx)
// 	if err != nil {
// 		return fmt.Errorf("failed to wait for config: %w", err)
// 	}

// 	// Wait for token
// 	_, err = ys.tokenManager.WaitForToken(ctx)
// 	if err != nil {
// 		return fmt.Errorf("failed to wait for token: %w", err)
// 	}

// 	return nil
// }

func (ys *YeastarService) WaitForInitialization(ctx context.Context) error {
	fmt.Println("[YeastarService] Starting initialization with Corteza triggers...")

	// Trigger config push from Corteza first
	fmt.Println("[YeastarService] Triggering config push from Corteza...")
	if err := ys.cortezaClient.TriggerConfigPush(); err != nil {
		return fmt.Errorf("failed to trigger config push: %w", err)
	}

	// Wait for config
	fmt.Println("[YeastarService] Waiting for config from Corteza...")
	config, err := ys.configManager.WaitForConfig(ctx)
	if err != nil {
		return fmt.Errorf("failed to wait for config: %w", err)
	}
	fmt.Println("[YeastarService]  Config received successfully")

	// Trigger token push from Corteza
	fmt.Println("[YeastarService] Triggering token push from Corteza...")
	if err := ys.cortezaClient.TriggerTokenPush(); err != nil {
		fmt.Printf("[YeastarService] Warning: failed to trigger token push: %v\n", err)
	}

	// Wait for token with timeout and fallback
	tokenCtx, tokenCancel := context.WithTimeout(ctx, 15*time.Second)
	defer tokenCancel()

	fmt.Println("[YeastarService] Waiting for token from Corteza...")
	token, err := ys.tokenManager.WaitForToken(tokenCtx)
	if err != nil || token == nil || token.AccessToken == "" {
		fmt.Println("[YeastarService] No valid token from Corteza, getting fresh one from Yeastar...")

		// Get fresh token directly from Yeastar
		freshToken, err := ys.tokenManager.GetNewToken(ctx, config)
		if err != nil {
			return fmt.Errorf("failed to get fresh token from Yeastar: %w", err)
		}

		fmt.Println("[YeastarService] Saving fresh token to Corteza...")
		if err := ys.cortezaClient.SaveToken(ctx, freshToken); err != nil {
			// Log warning but don't fail - we have a working token
			fmt.Printf("[YeastarService] Warning: failed to save token to Corteza: %v\n", err)
		}

		// Set the token directly since we just got it
		ys.tokenManager.SetToken(freshToken)
		fmt.Println("[YeastarService]  Fresh token obtained and set")
	} else {
		fmt.Println("[YeastarService]  Token received from Corteza")
	}

	// Verify we have a valid token
	if !ys.tokenManager.IsTokenValid() {
		return fmt.Errorf("no valid token available after all attempts")
	}

	fmt.Println("[YeastarService]  Initialization completed successfully")
	return nil
}

// EnsureValidToken ensures we have a valid token, refreshing if necessary
func (ys *YeastarService) EnsureValidToken(ctx context.Context) error {
	if ys.tokenManager.IsTokenValid() {
		return nil
	}

	config := ys.configManager.GetConfig()
	if config == nil {
		return fmt.Errorf("config not available")
	}

	var newToken *TokenResponse
	var err error

	// Try to refresh token first, if possible
	if ys.tokenManager.CanRefreshToken() {
		fmt.Println("Attempting to refresh token...")
		newToken, err = ys.tokenManager.RefreshToken(ctx, config)
		if err != nil {
			fmt.Printf("Token refresh failed: %v. Getting new token...\n", err)
			newToken, err = ys.tokenManager.GetNewToken(ctx, config)
		}
	} else {
		fmt.Println("Getting new token...")
		newToken, err = ys.tokenManager.GetNewToken(ctx, config)
	}

	if err != nil {
		return fmt.Errorf("failed to get valid token: %w", err)
	}

	// Save the new token to Corteza
	if err := ys.cortezaClient.SaveToken(ctx, newToken); err != nil {
		// Log error but don't fail the operation
		fmt.Printf("Warning: failed to save token to Corteza: %v\n", err)
	}

	// Reset token manager state and trigger Corteza to send the updated token
	ys.tokenManager.ResetTokenState()

	// Trigger Corteza to send the updated token back
	if err := ys.cortezaClient.TriggerTokenPush(); err != nil {
		// If trigger fails, fall back to using the token directly
		fmt.Printf("Warning: failed to trigger token push from Corteza: %v. Using token directly.\n", err)
		ys.tokenManager.SetToken(newToken)
		return nil
	}

	// Wait for the token to be received from Corteza
	fmt.Println("Waiting for updated token from Corteza...")
	_, err = ys.tokenManager.WaitForToken(ctx)
	if err != nil {
		// If waiting fails, fall back to using the token directly
		fmt.Printf("Warning: failed to receive token from Corteza: %v. Using token directly.\n", err)
		ys.tokenManager.SetToken(newToken)
	}

	return nil
}

// ListMethod performs API calls with automatic token management and retry logic
func (ys *YeastarService) ListMethod(ctx context.Context, endpoint string) ([]byte, error) {
	var lastErr error

	for attempt := 0; attempt <= ys.maxRetries; attempt++ {
		if attempt > 0 {
			delay := ys.baseRetryDelay * time.Duration(attempt)
			log.Printf("[Attempt %d] Waiting %s before retrying...", attempt, delay)
			select {
			case <-time.After(delay):
			case <-ctx.Done():
				log.Printf("[Attempt %d] Context done while waiting to retry: %v", attempt, ctx.Err())
				return nil, ctx.Err()
			}
		}

		log.Printf("[Attempt %d] Ensuring valid token for endpoint: %s", attempt, endpoint)
		if err := ys.EnsureValidToken(ctx); err != nil {
			lastErr = fmt.Errorf("failed to ensure valid token: %w", err)
			log.Printf("[Attempt %d] Error ensuring valid token: %v", attempt, err)
			continue
		}

		config := ys.configManager.GetConfig()
		token := ys.tokenManager.GetToken()

		if config == nil || token == nil {
			lastErr = fmt.Errorf("config or token not available")
			log.Printf("[Attempt %d] Config or token not available", attempt)
			continue
		}

		url := fmt.Sprintf("%s/openapi/v1.0/%s/list?access_token=%s",
			config.ApiBaseUrl,
			endpoint,
			token.AccessToken,
		)

		log.Printf("[Attempt %d] Making GET request to URL: %s", attempt, url)
		req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
		if err != nil {
			lastErr = fmt.Errorf("failed to create request: %w", err)
			log.Printf("[Attempt %d] Error creating request: %v", attempt, err)
			continue
		}

		resp, err := ys.httpClient.Do(req)
		if err != nil {
			lastErr = fmt.Errorf("API request failed: %w", err)
			log.Printf("[Attempt %d] API request failed: %v", attempt, err)
			continue
		}
		defer resp.Body.Close()

		body, err := io.ReadAll(resp.Body)
		if err != nil {
			lastErr = fmt.Errorf("failed to read response: %w", err)
			log.Printf("[Attempt %d] Failed to read response body: %v", attempt, err)
			continue
		}

		log.Printf("[Attempt %d] Response Status: %d, Body: %s", attempt, resp.StatusCode, string(body))

		if resp.StatusCode == http.StatusOK {
			log.Printf("[Attempt %d] Successful response received, length %d bytes", attempt, len(body))
			return body, nil
		}

		// Handle specific error codes
		if resp.StatusCode == http.StatusUnauthorized {
			log.Printf("[Attempt %d] Unauthorized response received, resetting token and retrying", attempt)
			ys.tokenManager.ResetTokenState()
			lastErr = fmt.Errorf("unauthorized access, will retry with new token")
			continue
		}

		lastErr = fmt.Errorf("unexpected status code %d from endpoint %s: %s",
			resp.StatusCode, endpoint, string(body))
		log.Printf("[Attempt %d] Unexpected status code: %d, response body: %s", attempt, resp.StatusCode, string(body))

		// For non-recoverable errors, don't retry
		if resp.StatusCode >= 400 && resp.StatusCode < 500 && resp.StatusCode != http.StatusUnauthorized {
			log.Printf("[Attempt %d] Non-recoverable client error, not retrying", attempt)
			break
		}
	}

	log.Printf("Failed to get data from endpoint %s after %d attempts: %v", endpoint, ys.maxRetries+1, lastErr)
	return nil, fmt.Errorf("failed after %d attempts: %w", ys.maxRetries+1, lastErr)
}

func (ys *YeastarService) GetRecordingsList(ctx context.Context) ([]Recording, error) {
	const endpoint = "recording"

	// log.Printf("[INFO] Fetching recordings list from endpoint: %s", endpoint)

	rawData, err := ys.ListMethod(ctx, endpoint)
	if err != nil {
		// log.Printf("[ERROR] Failed to fetch recordings: %v", err)
		return nil, fmt.Errorf("failed to fetch recordings: %w", err)
	}

	//log.Printf("[DEBUG] Raw response data: %s", string(rawData))

	var response struct {
		ErrCode     int         `json:"errcode"`
		ErrMsg      string      `json:"errmsg"`
		TotalNumber int         `json:"total_number"`
		Data        []Recording `json:"data"`
	}

	if err := json.Unmarshal(rawData, &response); err != nil {
		// log.Printf("[ERROR] Failed to unmarshal recordings: %v", err)
		return nil, fmt.Errorf("failed to unmarshal recordings: %w", err)
	}

	if response.ErrCode != 0 {
		// log.Printf("[ERROR] Recordings fetch failed: %s", response.ErrMsg)
		return nil, fmt.Errorf("recordings fetch failed: %s", response.ErrMsg)
	}

	log.Printf("[INFO] Successfully fetched %d recordings", len(response.Data))
	return response.Data, nil
}

func (ys *YeastarService) GetRecordingDownloadURL(ctx context.Context, recordingID int) (string, error) {
	// log.Printf("[INFO] Getting download URL for recording ID: %d", recordingID)

	if err := ys.EnsureValidToken(ctx); err != nil {
		log.Printf("[ERROR] Token validation failed: %v", err)
		return "", fmt.Errorf("failed to ensure valid token: %w", err)
	}

	config := ys.configManager.GetConfig()
	token := ys.tokenManager.GetToken()

	if config == nil || token == nil {
		log.Printf("[ERROR] Missing config or token")
		return "", fmt.Errorf("config or token not available")
	}

	url := fmt.Sprintf("%s/openapi/v1.0/recording/download?access_token=%s&id=%d",
		config.ApiBaseUrl,
		token.AccessToken,
		recordingID,
	)

	//log.Printf("[DEBUG] Constructed request URL: %s", url)

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		log.Printf("[ERROR] Failed to create HTTP request: %v", err)
		return "", fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := ys.httpClient.Do(req)
	if err != nil {
		log.Printf("[ERROR] HTTP request failed: %v", err)
		return "", fmt.Errorf("API request failed: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Printf("[ERROR] Failed to read response body: %v", err)
		return "", fmt.Errorf("failed to read response: %w", err)
	}

	// log.Printf("[DEBUG] Response body: %s", string(body))

	if resp.StatusCode != http.StatusOK {
		log.Printf("[ERROR] Unexpected status code: %d", resp.StatusCode)
		return "", fmt.Errorf("unexpected status code %d: %s", resp.StatusCode, string(body))
	}

	var downloadResp struct {
		ErrCode             int    `json:"errcode"`
		ErrMsg              string `json:"errmsg"`
		File                string `json:"file"`
		DownloadResourceURL string `json:"download_resource_url"`
	}

	if err := json.Unmarshal(body, &downloadResp); err != nil {
		// log.Printf("[ERROR] Failed to unmarshal download response: %v", err)
		return "", fmt.Errorf("failed to unmarshal download response: %w", err)
	}

	if downloadResp.ErrCode != 0 {
		// log.Printf("[ERROR] API returned error: %s", downloadResp.ErrMsg)
		return "", fmt.Errorf("download API error: %s", downloadResp.ErrMsg)
	}

	fullURL := config.ApiBaseUrl + downloadResp.DownloadResourceURL
	// log.Printf("[INFO] Successfully obtained download URL: %s", fullURL)

	return fullURL, nil
}

// SendDataToCorteza sends processed data to Corteza
func (ys *YeastarService) SendDataToCorteza(ctx context.Context, moduleName string, data interface{}) error {
	return ys.cortezaClient.SendData(ctx, moduleName, data)
}

func (ys *YeastarService) SearchMethod(ctx context.Context, endpoint string) ([]byte, error) {
	var lastErr error

	for attempt := 0; attempt <= ys.maxRetries; attempt++ {
		if attempt > 0 {
			delay := ys.baseRetryDelay * time.Duration(attempt)
			log.Printf("[Attempt %d] Waiting %s before retrying...", attempt, delay)
			select {
			case <-time.After(delay):
			case <-ctx.Done():
				log.Printf("[Attempt %d] Context done while waiting to retry: %v", attempt, ctx.Err())
				return nil, ctx.Err()
			}
		}

		log.Printf("[Attempt %d] Ensuring valid token for endpoint: %s", attempt, endpoint)
		if err := ys.EnsureValidToken(ctx); err != nil {
			lastErr = fmt.Errorf("failed to ensure valid token: %w", err)
			log.Printf("[Attempt %d] Error ensuring valid token: %v", attempt, err)
			continue
		}

		config := ys.configManager.GetConfig()
		token := ys.tokenManager.GetToken()

		if config == nil || token == nil {
			lastErr = fmt.Errorf("config or token not available")
			log.Printf("[Attempt %d] Config or token not available", attempt)
			continue
		}

		now := time.Now()
		startTime := now.Add(-15*time.Minute - 1*time.Hour).Format("02/01/2006 15:04:05")
		endTime := now.Add(15*time.Minute - 1*time.Hour).Format("02/01/2006 15:04:05")

		url := fmt.Sprintf("%s/openapi/v1.0/%s/search?access_token=%s&start_time=%s&end_time=%s",
			config.ApiBaseUrl,
			endpoint,
			token.AccessToken,
			url.QueryEscape(startTime),
			url.QueryEscape(endTime),
		)

		log.Printf("[Attempt %d] Making GET request to URL: %s", attempt, url)
		req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
		if err != nil {
			lastErr = fmt.Errorf("failed to create request: %w", err)
			log.Printf("[Attempt %d] Error creating request: %v", attempt, err)
			continue
		}

		resp, err := ys.httpClient.Do(req)
		if err != nil {
			lastErr = fmt.Errorf("API request failed: %w", err)
			log.Printf("[Attempt %d] API request failed: %v", attempt, err)
			continue
		}
		defer resp.Body.Close()

		body, err := io.ReadAll(resp.Body)
		if err != nil {
			lastErr = fmt.Errorf("failed to read response: %w", err)
			log.Printf("[Attempt %d] Failed to read response body: %v", attempt, err)
			continue
		}

		log.Printf("[Attempt %d] Response Status: %d, Body: %s", attempt, resp.StatusCode, string(body))

		if resp.StatusCode == http.StatusOK {
			log.Printf("[Attempt %d] Successful response received, length %d bytes", attempt, len(body))
			return body, nil
		}

		if resp.StatusCode == http.StatusUnauthorized {
			log.Printf("[Attempt %d] Unauthorized response received, resetting token and retrying", attempt)
			ys.tokenManager.ResetTokenState()
			lastErr = fmt.Errorf("unauthorized access, will retry with new token")
			continue
		}

		lastErr = fmt.Errorf("unexpected status code %d from endpoint %s: %s",
			resp.StatusCode, endpoint, string(body))
		log.Printf("[Attempt %d] Unexpected status code: %d, response body: %s", attempt, resp.StatusCode, string(body))

		if resp.StatusCode >= 400 && resp.StatusCode < 500 && resp.StatusCode != http.StatusUnauthorized {
			log.Printf("[Attempt %d] Non-recoverable client error, not retrying", attempt)
			break
		}
	}

	log.Printf("Failed to get data from endpoint %s after %d attempts: %v", endpoint, ys.maxRetries+1, lastErr)
	return nil, fmt.Errorf("failed after %d attempts: %w", ys.maxRetries+1, lastErr)
}
