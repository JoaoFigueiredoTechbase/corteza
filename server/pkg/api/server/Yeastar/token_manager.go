package Yeastar

import (
	"bytes"
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sync"
	"time"
)

// TokenManager handles token lifecycle with automatic refresh
type TokenManager struct {
	token     *TokenResponse
	mu        sync.RWMutex
	readyChan chan struct{}
	onceReady sync.Once
	isReady   bool
	client    *http.Client
}

// NewTokenManager creates a new token manager
func NewTokenManager() *TokenManager {
	return &TokenManager{
		readyChan: make(chan struct{}),
		client: &http.Client{
			Timeout: 30 * time.Second,
			Transport: &http.Transport{
				TLSClientConfig: &tls.Config{InsecureSkipVerify: true}, // DANGER! Only for development!
			},
		},
	}
}

// SetToken sets the token and marks it as ready
func (tm *TokenManager) SetToken(token *TokenResponse) {
	tm.mu.Lock()
	defer tm.mu.Unlock()

	tm.token = token
	tm.onceReady.Do(func() {
		tm.isReady = true
		close(tm.readyChan)
	})

	expiresIn := int64(token.AccessTokenExpireTime) - time.Now().Unix()
	fmt.Printf("✅ Token set. Access token expires in: %ds\n", expiresIn)
}

func (tm *TokenManager) ResetTokenState() {
	tm.mu.Lock()
	defer tm.mu.Unlock()

	tm.token = nil
	tm.isReady = false
	tm.readyChan = make(chan struct{})
	tm.onceReady = sync.Once{}
}

// GetToken returns the current token
func (tm *TokenManager) GetToken() *TokenResponse {
	tm.mu.RLock()
	defer tm.mu.RUnlock()
	return tm.token
}

// WaitForToken waits for token to be available
func (tm *TokenManager) WaitForToken(ctx context.Context) (*TokenResponse, error) {
	select {
	case <-tm.readyChan:
		return tm.GetToken(), nil
	case <-ctx.Done():
		return nil, ctx.Err()
	}
}

// IsReady returns true if token is ready
func (tm *TokenManager) IsReady() bool {
	tm.mu.RLock()
	defer tm.mu.RUnlock()
	return tm.isReady
}

// IsTokenValid checks if the current token is still valid
func (tm *TokenManager) IsTokenValid() bool {
	tm.mu.RLock()
	defer tm.mu.RUnlock()

	if tm.token == nil {
		return false
	}

	return time.Now().Unix() < int64(tm.token.AccessTokenExpireTime)
}

// CanRefreshToken checks if the token can be refreshed
func (tm *TokenManager) CanRefreshToken() bool {
	tm.mu.RLock()
	defer tm.mu.RUnlock()

	if tm.token == nil {
		return false
	}

	return time.Now().Unix() < int64(tm.token.RefreshTokenExpireTime)
}

func (tm *TokenManager) NeedsRefresh() bool {
	tm.mu.RLock()
	defer tm.mu.RUnlock()

	if tm.token == nil {
		return true
	}
	// Give a small buffer before expiry to avoid race conditions
	bufferSeconds := int64(60)
	return time.Now().Unix() >= int64(tm.token.AccessTokenExpireTime)-bufferSeconds
}

func (tm *TokenManager) GetNewToken(ctx context.Context, cfg *Config) (*TokenResponse, error) {
	fmt.Println("🆕 Starting request for new token")

	url := fmt.Sprintf("%s/openapi/v1.0/get_token", cfg.ApiBaseUrl)
	fmt.Printf("🌍 Token request URL: %s\n", url)

	payload := map[string]string{
		"username": cfg.ApiUserName,
		"password": cfg.ApiSecret,
	}
	fmt.Printf("🔐 Credentials - Username: %s\n", cfg.ApiUserName)

	jsonPayload, err := json.Marshal(payload)
	if err != nil {
		fmt.Printf("❌ Failed to marshal token request: %v\n", err)
		return nil, fmt.Errorf("failed to marshal token request: %w", err)
	}
	fmt.Printf("📤 Payload to send: %s\n", string(jsonPayload))

	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(jsonPayload))
	if err != nil {
		fmt.Printf("❌ Failed to create token request: %v\n", err)
		return nil, fmt.Errorf("failed to create token request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	fmt.Println("📡 Sending token request")
	resp, err := tm.client.Do(req)
	if err != nil {
		fmt.Printf("❌ Failed to send token request: %v\n", err)
		return nil, fmt.Errorf("failed to request token: %w", err)
	}
	defer resp.Body.Close()

	fmt.Printf("📥 Response received with status code: %d\n", resp.StatusCode)

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		fmt.Printf("❌ Token request failed: %d - %s\n", resp.StatusCode, string(body))
		return nil, fmt.Errorf("token request failed: %d, body: %s", resp.StatusCode, string(body))
	}

	var tokenResp TokenResponse
	if err := json.NewDecoder(resp.Body).Decode(&tokenResp); err != nil {
		fmt.Printf("❌ Failed to decode token response: %v\n", err)
		return nil, fmt.Errorf("failed to decode token response: %w", err)
	}
	fmt.Printf("✅ Token received: %+v\n", tokenResp)

	// Convert relative times to absolute timestamps
	now := float64(time.Now().Unix())
	tokenResp.AccessTokenExpireTime = now + tokenResp.AccessTokenExpireTime
	tokenResp.RefreshTokenExpireTime = now + tokenResp.RefreshTokenExpireTime

	fmt.Printf("⏳ Access token expires at: %f\n", tokenResp.AccessTokenExpireTime)
	fmt.Printf("🔄 Refresh token expires at: %f\n", tokenResp.RefreshTokenExpireTime)

	return &tokenResp, nil
}

func (tm *TokenManager) RefreshToken(ctx context.Context, cfg *Config) (*TokenResponse, error) {
	fmt.Println("🔄 Starting token refresh")

	currentToken := tm.GetToken()
	if currentToken == nil {
		fmt.Println("❌ No current token found to refresh")
		return nil, fmt.Errorf("no current token to refresh")
	}

	fmt.Printf("🔑 Current refresh token: %s\n", currentToken.RefreshToken)

	url := fmt.Sprintf("%s/openapi/v1.0/refresh_token", cfg.ApiBaseUrl)
	fmt.Printf("🌍 Refresh URL: %s\n", url)

	payload := map[string]string{
		"refresh_token": currentToken.RefreshToken,
	}

	jsonPayload, err := json.Marshal(payload)
	if err != nil {
		fmt.Printf("❌ Failed to marshal refresh request: %v\n", err)
		return nil, fmt.Errorf("failed to marshal refresh request: %w", err)
	}
	fmt.Printf("📤 Payload to send: %s\n", string(jsonPayload))

	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(jsonPayload))
	if err != nil {
		fmt.Printf("❌ Failed to create refresh request: %v\n", err)
		return nil, fmt.Errorf("failed to create refresh request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	fmt.Println("📡 Sending token refresh request")
	resp, err := tm.client.Do(req)
	if err != nil {
		fmt.Printf("❌ Failed to send refresh request: %v\n", err)
		return nil, fmt.Errorf("failed to refresh token: %w", err)
	}
	defer resp.Body.Close()

	fmt.Printf("📥 Received response with status code: %d\n", resp.StatusCode)

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		fmt.Printf("❌ Refresh failed with status %d: %s\n", resp.StatusCode, string(body))
		return nil, fmt.Errorf("token refresh failed: %d, body: %s", resp.StatusCode, string(body))
	}

	var tokenResp TokenResponse
	if err := json.NewDecoder(resp.Body).Decode(&tokenResp); err != nil {
		fmt.Printf("❌ Failed to decode token response: %v\n", err)
		return nil, fmt.Errorf("failed to decode refresh response: %w", err)
	}

	if tokenResp.AccessToken == "" || tokenResp.RefreshToken == "" {
		fmt.Println("❌ Received empty access or refresh token after refresh — rejecting")
		return nil, fmt.Errorf("invalid refresh response: missing token fields")
	}

	now := float64(time.Now().Unix())
	tokenResp.AccessTokenExpireTime = now + tokenResp.AccessTokenExpireTime
	tokenResp.RefreshTokenExpireTime = now + tokenResp.RefreshTokenExpireTime

	fmt.Println("✅ Token refresh successful")
	fmt.Printf("⏳ Access token expires at: %.2f\n", tokenResp.AccessTokenExpireTime)
	fmt.Printf("🔄 Refresh token expires at: %2f\n", tokenResp.RefreshTokenExpireTime)

	return &tokenResp, nil
}
