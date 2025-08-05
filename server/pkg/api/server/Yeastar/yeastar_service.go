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

func (ys *YeastarService) EnsureValidToken(ctx context.Context) error {
	if ys.tokenManager.IsTokenValid() {
		return nil
	}

	config := ys.configManager.GetConfig()
	if config == nil {
		return fmt.Errorf("config not available")
	}

	var token *TokenResponse
	var err error

	if ys.tokenManager.CanRefreshToken() {
		fmt.Println("[EnsureValidToken] Refreshing token...")
		token, err = ys.tokenManager.RefreshToken(ctx, config)
		if err != nil {
			fmt.Printf("[EnsureValidToken] Refresh failed: %v. Getting new token...\n", err)
			token, err = ys.tokenManager.GetNewToken(ctx, config)
		}
	} else {
		fmt.Println("[EnsureValidToken] Getting new token...")
		token, err = ys.tokenManager.GetNewToken(ctx, config)
	}

	if err != nil {
		return fmt.Errorf("failed to acquire valid token: %w", err)
	}

	// Save to Corteza (optional, for syncing)
	if err := ys.cortezaClient.SaveToken(ctx, token); err != nil {
		fmt.Printf("[EnsureValidToken] Warning: failed to save token to Corteza: %v\n", err)
	}

	// Use token directly
	ys.tokenManager.SetToken(token)

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

func (ys *YeastarService) SearchExtension(ctx context.Context, searchValue string) ([]byte, error) {
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

		url := fmt.Sprintf("%s/openapi/v1.0/extension/search?access_token=%s&search_value=%s",
			config.ApiBaseUrl,
			token.AccessToken,
			searchValue,
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

		log.Printf("[Attempt %d] Unexpected status code: %d, response body: %s", attempt, resp.StatusCode, string(body))

		if resp.StatusCode >= 400 && resp.StatusCode < 500 && resp.StatusCode != http.StatusUnauthorized {
			log.Printf("[Attempt %d] Non-recoverable client error, not retrying", attempt)
			break
		}
	}

	log.Printf("Failed to get data  after %d attempts: %v", ys.maxRetries+1, lastErr)
	return nil, fmt.Errorf("failed after %d attempts: %w", ys.maxRetries+1, lastErr)
}

func SearchNewCDR(baseUrl, uid string) error {
	service, ctx, cancel, err := setupSyncService(baseUrl)
	if err != nil {
		return err
	}
	defer cancel()

	if err := setupAuth(ctx, service); err != nil {
		return err
	}

	rawCDRsData, err := service.SearchMethod(ctx, "cdr")
	if err != nil {
		return fmt.Errorf("failed to search cdrs: %w", err)
	}

	// log.Printf("Raw CDRs data length: %d", len(rawCDRsData))
	// log.Printf("Raw CDRs data: %+v", rawCDRsData)

	cdrs, err := processCDRsData(service, rawCDRsData)
	if err != nil {
		return fmt.Errorf("failed to process cdrs: %w", err)
	}

	// log.Printf("Processed CDRs count: %d", len(cdrs))
	for i, cdr := range cdrs {
		log.Printf("CDR[%d]: UID=%s", i, cdr.UID)
	}
	// log.Printf("Looking for UID: %s", uid)

	results, err := findCDRsByUID(cdrs, uid)
	if err != nil {
		log.Printf("Error: %v\n", err)
	} else {
		for _, cdr := range results {
			fmt.Printf("Found CDR: %+v\n", cdr)
		}
	}

	if err := service.SendDataToCorteza(ctx, "cdr", results); err != nil {
		return fmt.Errorf("failed to send cdrs to Corteza: %w", err)
	}

	fmt.Println("cdrs processed and sent to Corteza successfully!")
	return nil
}

func SearchExtensionContext(baseUrl, searchValue string) (Agent, error) {
	service, ctx, cancel, err := setupSyncService(baseUrl)
	if err != nil {
		return Agent{}, err
	}
	defer cancel()

	if err := setupAuth(ctx, service); err != nil {
		return Agent{}, err
	}

	rawExtension, err := service.SearchExtension(ctx, searchValue)
	if err != nil {
		return Agent{}, fmt.Errorf("failed to search cdrs: %w", err)
	}

	agents, err := processAgentsData(rawExtension)
	if err != nil {
		return Agent{}, fmt.Errorf("failed to process cdrs: %w", err)
	}

	if len(agents) == 0 {
		return Agent{}, fmt.Errorf("no agents found")
	}

	return agents[0], nil
}
