package Yeastar

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

// Global managers - initialized in main.go or init function
var (
	GlobalConfigManager *ConfigManager
	GlobalTokenManager  *TokenManager
)

// ConfigCallbackHandler handles configuration callbacks from Corteza
func ConfigCallbackHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	defer r.Body.Close()

	var cfg Config
	if err := json.NewDecoder(r.Body).Decode(&cfg); err != nil {
		http.Error(w, fmt.Sprintf("Invalid config JSON: %v", err), http.StatusBadRequest)
		return
	}

	// Validate required fields
	if cfg.ApiBaseUrl == "" || cfg.ApiUserName == "" || cfg.ApiSecret == "" {
		http.Error(w, "Missing required config fields", http.StatusBadRequest)
		return
	}

	GlobalConfigManager.SetConfig(&cfg)
	fmt.Printf("Received config from Corteza: BaseURL=%s, Username=%s\n", cfg.ApiBaseUrl, cfg.ApiUserName)

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"status": "success"})
}

// TokenCallbackHandler handles token callbacks from Corteza
func TokenCallbackHandler(w http.ResponseWriter, r *http.Request) {
	fmt.Println("TokenCallbackHandler invoked")

	if r.Method != http.MethodPost {
		fmt.Printf("Invalid method: %s\n", r.Method)
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	defer r.Body.Close()

	// Read the raw body for logging
	bodyBytes, err := io.ReadAll(r.Body)
	if err != nil {
		fmt.Printf("Failed to read request body: %v\n", err)
		http.Error(w, "Failed to read request body", http.StatusBadRequest)
		return
	}
	fmt.Printf("Raw request body: %s\n", string(bodyBytes))

	// Decode JSON
	var token TokenResponse
	if err := json.Unmarshal(bodyBytes, &token); err != nil {
		fmt.Printf("Invalid token JSON: %v\n", err)
		http.Error(w, fmt.Sprintf("Invalid token JSON: %v", err), http.StatusBadRequest)
		return
	}
	fmt.Printf("Decoded token: %+v\n", token)

	// Validate token fields
	now := float64(time.Now().Unix())
	expiresIn := token.AccessTokenExpireTime - now
	refreshExpiresIn := token.RefreshTokenExpireTime - now

	if token.AccessToken == "" || token.RefreshToken == "" ||
		token.AccessTokenExpireTime == 0 || token.RefreshTokenExpireTime == 0 {
		fmt.Println("Missing required token fields")
		http.Error(w, "Missing required token fields", http.StatusBadRequest)
		return
	}

	if expiresIn <= 0 {
		fmt.Printf("Access token is already expired (expiresIn: %.0fs)\n", expiresIn)
		http.Error(w, "Access token is already expired", http.StatusBadRequest)
		return
	}

	if refreshExpiresIn <= 0 {
		fmt.Printf("Refresh token is already expired (refreshExpiresIn: %.0fs)\n", refreshExpiresIn)
		http.Error(w, "Refresh token is already expired", http.StatusBadRequest)
		return
	}

	fmt.Println("Valid token received, updating global token manager")
	GlobalTokenManager.SetToken(&token)

	fmt.Printf("Access token expires at: %d\n", token.AccessTokenExpireTime)

	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(map[string]string{"status": "success"}); err != nil {
		fmt.Printf("Failed to encode success response: %v\n", err)
	}
}
