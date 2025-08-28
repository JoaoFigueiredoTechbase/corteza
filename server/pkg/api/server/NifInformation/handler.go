package nifinformation

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"strings"
	"time"
)

const (
	APIBaseURL     = "https://www.nif.pt"
	RequestTimeout = 30 * time.Second
	APIRateLimit   = time.Minute
)

func HandleClientInformationSearch(w http.ResponseWriter, r *http.Request) {
	usage, err := loadUsage()
	if err != nil {
		log.Printf("ERROR: failed to load usage: %v", err)
		http.Error(w, "Internal error", http.StatusInternalServerError)
		return
	}

	limits := RateLimits{
		Month:  1000,
		Day:    100,
		Hour:   10,
		Minute: 1,
	}

	if !checkAndUpdateQuota(&usage, limits) {
		http.Error(w, "API request limit exceeded", http.StatusTooManyRequests)
		return
	}

	if err := saveUsage(usage); err != nil {
		log.Printf("WARN: failed to save usage: %v", err)
	}

	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Minute)
	defer cancel()

	log.Println("Received Client Information Search Request")

	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var payload ClientInformation
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		log.Printf("ERROR: Failed to decode request body: %v", err)
		http.Error(w, "Invalid JSON payload", http.StatusBadRequest)
		return
	}

	if strings.TrimSpace(payload.ApiKey) == "" {
		log.Println("ERROR: Missing API key")
		http.Error(w, "API key is required", http.StatusBadRequest)
		return
	}

	var query string
	isNameQuery := false

	payload.ClientNif = strings.TrimSpace(payload.ClientNif)
	payload.ClientName = strings.TrimSpace(payload.ClientName)

	if payload.ClientNif != "" {
		if !validateNif(payload.ClientNif) {
			log.Printf("ERROR: Invalid NIF format: %s", payload.ClientNif)
			http.Error(w, "NIF must be exactly 9 digits", http.StatusBadRequest)
			return
		}
		query = payload.ClientNif
	} else if payload.ClientName != "" {
		query = sanitizeQuery(payload.ClientName)
		if query == "" {
			log.Println("ERROR: Client name contains no valid characters")
			http.Error(w, "Client name contains no valid characters", http.StatusBadRequest)
			return
		}
		isNameQuery = true
	} else {
		log.Println("ERROR: Neither client_nif nor client_name provided")
		http.Error(w, "Either client_nif (9 digits) or client_name is required", http.StatusBadRequest)
		return
	}

	apiURL := fmt.Sprintf("%s/?json=1&q=%s&key=%s", APIBaseURL, url.QueryEscape(query), payload.ApiKey)
	log.Printf("Making API request to: %s", strings.Replace(apiURL, payload.ApiKey, "[REDACTED]", 1))

	apiResp, err := makeAPIRequest(ctx, apiURL)
	if err != nil {
		log.Printf("ERROR: API request failed: %v", err)
		http.Error(w, "Failed to query NIF API", http.StatusInternalServerError)
		return
	}

	var results []NifApiResponse
	for _, raw := range apiResp.Records {
		record, err := parseRecord(raw)
		if err != nil {
			log.Printf("WARN: Failed to parse record: %v", err)
			continue
		}
		if record.Nif != 0 {
			results = append(results, record)
		}
	}

	if len(results) == 0 {
		log.Printf("INFO: No records found for query: %s", query)
		http.Error(w, "No records found", http.StatusNotFound)
		return
	}

	if isNameQuery && len(results) != 1 {
		best, ok := pickBestMatch(results, payload.ClientName)
		if !ok {
			log.Printf("INFO: No suitable match found for name: %s", payload.ClientName)
			http.Error(w, "No suitable match found", http.StatusNotFound)
			return
		}

		log.Printf("INFO: Best match found - NIF: %d, Title: %s", best.Nif, best.Title)
		log.Printf("INFO: Waiting %v due to API rate limiting before fetching full details", APIRateLimit)

		select {
		case <-time.After(APIRateLimit):
			// Continue after delay
		case <-ctx.Done():
			log.Println("ERROR: Request cancelled during rate limit wait")
			http.Error(w, "Request timeout", http.StatusRequestTimeout)
			return
		}

		fullRecord, err := fetchFullInfo(ctx, best.Nif, payload.ApiKey)
		if err != nil {
			log.Printf("ERROR: Failed to fetch full info for NIF %d: %v", best.Nif, err)
			http.Error(w, "Failed to fetch complete information", http.StatusInternalServerError)
			return
		}

		log.Printf("INFO: Successfully retrieved full information for NIF: %d", fullRecord.Nif)
		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(fullRecord); err != nil {
			log.Printf("ERROR: Failed to encode full record: %v", err)
			http.Error(w, "Failed to encode response", http.StatusInternalServerError)
		}
		return
	}

	log.Printf("INFO: Returning %d records for NIF query", len(results))
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(results[0]); err != nil {
		log.Printf("ERROR: Failed to encode response: %v", err)
		http.Error(w, "Failed to encode response", http.StatusInternalServerError)
	}
}
func HandleTest(w http.ResponseWriter, r *http.Request) {
	data := map[string]interface{}{
		"nif":     513602011,
		"title":   "Mendes L. It & Communications, Unipessoal Lda",
		"address": "Rua Padre Francisco Rodrigues, Nº 2250",
		"pc4":     "4800",
		"pc3":     "606",
		"city":    "Prazins Santa Eufémia",
		"activity": "Actividades de serviços de consultoria e formação em engenharia de comunicações e informática. " +
			"Comércio a retalho e por grosso de equipamentos de comunicações e informática. " +
			"Implementação de redes de dados e informática.",
		"cae":     []string{"62020", "47420", "47410"},
		"email":   "geral@techbase.pt",
		"phone":   "220035908",
		"website": "www.techbase.pt",
		"fax":     "220035908",
		"region":  "Braga",
		"county":  "Guimarães",
		"parish":  "Santa Eufémia Prazins",
		"racius":  "https://www.racius.com/mendes-l-it-communications-unipessoal-lda/",
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(data)
}
