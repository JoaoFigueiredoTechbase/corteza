package Yeastar

import (
	"encoding/json"
	"fmt"
	"net/http"
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
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	defer r.Body.Close()

	var token TokenResponse
	if err := json.NewDecoder(r.Body).Decode(&token); err != nil {
		http.Error(w, fmt.Sprintf("Invalid token JSON: %v", err), http.StatusBadRequest)
		return
	}

	// Validate token
	if token.AccessToken == "" || token.RefreshToken == "" ||
		token.AccessTokenExpireTime == 0 || token.RefreshTokenExpireTime == 0 {
		http.Error(w, "Missing required token fields", http.StatusBadRequest)
		return
	}

	GlobalTokenManager.SetToken(&token)
	fmt.Printf("Received token from Corteza: expires at %d\n", token.AccessTokenExpireTime)

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"status": "success"})
}
