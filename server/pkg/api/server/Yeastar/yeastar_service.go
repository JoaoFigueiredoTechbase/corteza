package Yeastar

import (
	"context"
	"crypto/tls"
	"fmt"
	"io"
	"net/http"
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
func (ys *YeastarService) WaitForInitialization(ctx context.Context) error {
	// Wait for config
	_, err := ys.configManager.WaitForConfig(ctx)
	if err != nil {
		return fmt.Errorf("failed to wait for config: %w", err)
	}

	// Wait for token
	_, err = ys.tokenManager.WaitForToken(ctx)
	if err != nil {
		return fmt.Errorf("failed to wait for token: %w", err)
	}

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
			select {
			case <-time.After(delay):
			case <-ctx.Done():
				return nil, ctx.Err()
			}
		}

		// Ensure we have a valid token
		if err := ys.EnsureValidToken(ctx); err != nil {
			lastErr = fmt.Errorf("failed to ensure valid token: %w", err)
			continue
		}

		config := ys.configManager.GetConfig()
		token := ys.tokenManager.GetToken()

		if config == nil || token == nil {
			lastErr = fmt.Errorf("config or token not available")
			continue
		}

		url := fmt.Sprintf("%s/openapi/v1.0/%s/list?access_token=%s",
			config.ApiBaseUrl,
			endpoint,
			token.AccessToken,
		)

		req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
		if err != nil {
			lastErr = fmt.Errorf("failed to create request: %w", err)
			continue
		}

		resp, err := ys.httpClient.Do(req)
		if err != nil {
			lastErr = fmt.Errorf("API request failed: %w", err)
			continue
		}
		defer resp.Body.Close()

		body, err := io.ReadAll(resp.Body)
		if err != nil {
			lastErr = fmt.Errorf("failed to read response: %w", err)
			continue
		}

		if resp.StatusCode == http.StatusOK {
			return body, nil
		}

		// Handle specific error codes
		if resp.StatusCode == http.StatusUnauthorized {
			// Token might be invalid, reset and force refresh on next attempt
			fmt.Println("Received unauthorized response, resetting token...")
			ys.tokenManager.ResetTokenState()
			lastErr = fmt.Errorf("unauthorized access, will retry with new token")
			continue
		}

		lastErr = fmt.Errorf("unexpected status code %d from endpoint %s: %s",
			resp.StatusCode, endpoint, string(body))

		// For non-recoverable errors, don't retry
		if resp.StatusCode >= 400 && resp.StatusCode < 500 && resp.StatusCode != http.StatusUnauthorized {
			break
		}
	}

	return nil, fmt.Errorf("failed after %d attempts: %w", ys.maxRetries+1, lastErr)
}

// SendDataToCorteza sends processed data to Corteza
func (ys *YeastarService) SendDataToCorteza(ctx context.Context, moduleName string, data interface{}) error {
	return ys.cortezaClient.SendData(ctx, moduleName, data)
}
