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

	return time.Now().Unix() < tm.token.AccessTokenExpireTime
}

// CanRefreshToken checks if the token can be refreshed
func (tm *TokenManager) CanRefreshToken() bool {
	tm.mu.RLock()
	defer tm.mu.RUnlock()

	if tm.token == nil {
		return false
	}

	return time.Now().Unix() < tm.token.RefreshTokenExpireTime
}

// GetNewToken obtains a new token from the API
func (tm *TokenManager) GetNewToken(ctx context.Context, cfg *Config) (*TokenResponse, error) {
	url := fmt.Sprintf("%s/openapi/v1.0/get_token", cfg.ApiBaseUrl)

	payload := map[string]string{
		"username": cfg.ApiUserName,
		"password": cfg.ApiSecret,
	}

	jsonPayload, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal token request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(jsonPayload))
	if err != nil {
		return nil, fmt.Errorf("failed to create token request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := tm.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to request token: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("token request failed: %d, body: %s", resp.StatusCode, string(body))
	}

	var tokenResp TokenResponse
	if err := json.NewDecoder(resp.Body).Decode(&tokenResp); err != nil {
		return nil, fmt.Errorf("failed to decode token response: %w", err)
	}

	// Convert relative times to absolute timestamps
	now := time.Now().Unix()
	tokenResp.AccessTokenExpireTime = now + tokenResp.AccessTokenExpireTime
	tokenResp.RefreshTokenExpireTime = now + tokenResp.RefreshTokenExpireTime

	return &tokenResp, nil
}
func (tm *TokenManager) RefreshToken(ctx context.Context, cfg *Config) (*TokenResponse, error) {
	currentToken := tm.GetToken()
	if currentToken == nil {
		return nil, fmt.Errorf("no current token to refresh")
	}

	url := fmt.Sprintf("%s/openapi/v1.0/refresh_token", cfg.ApiBaseUrl)

	payload := map[string]string{
		"refresh_token": currentToken.RefreshToken,
	}

	jsonPayload, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal refresh request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(jsonPayload))
	if err != nil {
		return nil, fmt.Errorf("failed to create refresh request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := tm.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to refresh token: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("token refresh failed: %d, body: %s", resp.StatusCode, string(body))
	}

	var tokenResp TokenResponse
	if err := json.NewDecoder(resp.Body).Decode(&tokenResp); err != nil {
		return nil, fmt.Errorf("failed to decode refresh response: %w", err)
	}

	// Convert relative times to absolute timestamps
	now := time.Now().Unix()
	tokenResp.AccessTokenExpireTime = now + tokenResp.AccessTokenExpireTime
	tokenResp.RefreshTokenExpireTime = now + tokenResp.RefreshTokenExpireTime

	return &tokenResp, nil
}
