package nifinformation

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"regexp"
	"strings"
	"time"
)

const (
	APIBaseURL     = "https://www.nif.pt" // Changed to HTTPS if supported, fallback to HTTP if needed
	RequestTimeout = 30 * time.Second
	APIRateLimit   = time.Minute // One request per minute policy
)

type ClientInformation struct {
	ClientName string `json:"client_name"`
	ClientNif  string `json:"client_nif"`
	ApiKey     string `json:"api_key"`
}

type ApiResponse struct {
	Result  string                     `json:"result"`
	Records map[string]json.RawMessage `json:"records"`
}

type SearchResponse struct {
	Result  string                     `json:"result"`
	Records map[string]json.RawMessage `json:"records"`
}

type LightweightRecord struct {
	Nif    int    `json:"nif"`
	Title  string `json:"title"`
	City   string `json:"city"`
	Pc4    string `json:"pc4"`
	Pc3    string `json:"pc3"`
	Racius string `json:"racius"`
	Url    string `json:"seo_url"`
}

type NifApiResponse struct {
	Nif        int      `json:"nif"`
	Title      string   `json:"title"`
	Address    string   `json:"address"`
	Pc4        string   `json:"pc4"`
	Pc3        string   `json:"pc3"`
	City       string   `json:"city"`
	Activity   string   `json:"activity"`
	CaeList    []string `json:"cae"`
	Email      string   `json:"email"`
	Phone      string   `json:"phone"`
	Website    string   `json:"website"`
	Fax        string   `json:"fax"`
	Region     string   `json:"region"`
	County     string   `json:"county"`
	Parish     string   `json:"parish"`
	RaciusLink string   `json:"racius"`
}

// validateNif validates that NIF is exactly 9 digits
func validateNif(nif string) bool {
	if len(nif) != 9 {
		return false
	}
	matched, _ := regexp.MatchString(`^\d{9}$`, nif)
	return matched
}

// sanitizeQuery removes potentially harmful characters from query string
func sanitizeQuery(query string) string {
	// Remove any characters that might cause issues in URL
	query = strings.TrimSpace(query)
	// Keep only alphanumeric, spaces, and common business name characters
	reg := regexp.MustCompile(`[^a-zA-Z0-9\s\-.,&()]`)
	return reg.ReplaceAllString(query, "")
}

// createHTTPClient creates an HTTP client with timeout
func createHTTPClient() *http.Client {
	return &http.Client{
		Timeout: RequestTimeout,
	}
}

// makeAPIRequest makes a request to the NIF API with proper error handling
func makeAPIRequest(ctx context.Context, url string) (*ApiResponse, error) {
	client := createHTTPClient()

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("creating request: %w", err)
	}

	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("making HTTP request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("API returned status %d", resp.StatusCode)
	}

	var apiResp ApiResponse
	if err := json.NewDecoder(resp.Body).Decode(&apiResp); err != nil {
		return nil, fmt.Errorf("decoding JSON response: %w", err)
	}

	return &apiResp, nil
}

// func parseCandidates(respBody []byte) ([]NifApiResponse, error) {
// 	var searchResp SearchResponse
// 	if err := json.Unmarshal(respBody, &searchResp); err != nil {
// 		return nil, fmt.Errorf("unmarshaling search response: %w", err)
// 	}

// 	var candidates []NifApiResponse
// 	for _, raw := range searchResp.Records {
// 		var lw LightweightRecord
// 		if err := json.Unmarshal(raw, &lw); err != nil {
// 			log.Printf("WARN: Failed to unmarshal lightweight record: %v", err)
// 			continue // skip bad entries
// 		}
// 		if lw.Nif == 0 {
// 			continue // skip ones without nif
// 		}

// 		candidates = append(candidates, NifApiResponse{
// 			Nif:        lw.Nif,
// 			Title:      lw.Title,
// 			City:       lw.City,
// 			Pc4:        lw.Pc4,
// 			Pc3:        lw.Pc3,
// 			RaciusLink: lw.Racius,
// 		})
// 	}
// 	return candidates, nil
// }

func pickBestMatch(records []NifApiResponse, query string) (NifApiResponse, bool) {
	if len(records) == 0 {
		return NifApiResponse{}, false
	}

	query = strings.ToLower(strings.TrimSpace(query))
	queryWords := strings.Fields(query)

	bestScore := -1
	var best NifApiResponse
	found := false

	for _, rec := range records {
		title := strings.ToLower(strings.TrimSpace(rec.Title))
		if title == "" {
			continue
		}

		score := 0

		// Exact match gets highest score
		if title == query {
			return rec, true
		}

		// Exact substring match
		if strings.Contains(title, query) {
			score += 10
		}

		// Starts with query
		if strings.HasPrefix(title, query) {
			score += 7
		}

		// Ends with query
		if strings.HasSuffix(title, query) {
			score += 5
		}

		// Word-by-word scoring with order preservation
		titleWords := strings.Fields(title)
		wordMatches := 0
		lastIndex := -1
		inOrder := true

		for _, queryWord := range queryWords {
			for i, titleWord := range titleWords {
				if strings.Contains(titleWord, queryWord) {
					wordMatches++
					if lastIndex != -1 && i < lastIndex {
						inOrder = false
					}
					lastIndex = i
					break
				}
			}
		}

		// Bonus for word matches
		score += wordMatches * 2

		// Bonus for maintaining word order
		if inOrder && wordMatches > 1 {
			score += 3
		}

		// Bonus for matching all query words
		if wordMatches == len(queryWords) {
			score += 5
		}

		if score > bestScore {
			bestScore = score
			best = rec
			found = true
		}
	}

	return best, found
}

// parseRecord helper to map one record with better error handling
func parseRecord(raw json.RawMessage) (NifApiResponse, error) {
	var tmp struct {
		Nif      int      `json:"nif"`
		Title    string   `json:"title"`
		Address  string   `json:"address"`
		Pc4      string   `json:"pc4"`
		Pc3      string   `json:"pc3"`
		City     string   `json:"city"`
		Activity string   `json:"activity"`
		CaeList  []string `json:"cae"`
		Contacts struct {
			Email   string `json:"email"`
			Phone   string `json:"phone"`
			Website string `json:"website"`
			Fax     string `json:"fax"`
		} `json:"contacts"`
		Geo struct {
			Region string `json:"region"`
			County string `json:"county"`
			Parish string `json:"parish"`
		} `json:"geo"`
		Racius string `json:"racius"`
	}

	if err := json.Unmarshal(raw, &tmp); err != nil {
		return NifApiResponse{}, fmt.Errorf("unmarshaling record: %w", err)
	}

	return NifApiResponse{
		Nif:        tmp.Nif,
		Title:      tmp.Title,
		Address:    tmp.Address,
		Pc4:        tmp.Pc4,
		Pc3:        tmp.Pc3,
		City:       tmp.City,
		Activity:   tmp.Activity,
		CaeList:    tmp.CaeList,
		Email:      tmp.Contacts.Email,
		Phone:      tmp.Contacts.Phone,
		Website:    tmp.Contacts.Website,
		Fax:        tmp.Contacts.Fax,
		Region:     tmp.Geo.Region,
		County:     tmp.Geo.County,
		Parish:     tmp.Geo.Parish,
		RaciusLink: tmp.Racius,
	}, nil
}

func fetchFullInfo(ctx context.Context, nif int, apiKey string) (NifApiResponse, error) {
	url := fmt.Sprintf("%s/?json=1&q=%d&key=%s", APIBaseURL, nif, apiKey)

	apiResp, err := makeAPIRequest(ctx, url)
	if err != nil {
		return NifApiResponse{}, fmt.Errorf("API request failed: %w", err)
	}

	// Assuming only 1 record for a specific NIF
	for _, raw := range apiResp.Records {
		return parseRecord(raw)
	}

	return NifApiResponse{}, fmt.Errorf("no record found for NIF %d", nif)
}

func HandleClientInformationSearch(w http.ResponseWriter, r *http.Request) {
	// Create context with timeout for the entire request
	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Minute) // Allow time for rate limiting
	defer cancel()

	log.Println("Received Client Information Search Request")

	// Validate HTTP method
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

	// Validate API key
	if strings.TrimSpace(payload.ApiKey) == "" {
		log.Println("ERROR: Missing API key")
		http.Error(w, "API key is required", http.StatusBadRequest)
		return
	}

	// Determine query type and validate input
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

	// Make initial API request
	url := fmt.Sprintf("%s/?json=1&q=%s&key=%s", APIBaseURL, query, payload.ApiKey)
	log.Printf("Making API request to: %s", strings.Replace(url, payload.ApiKey, "[REDACTED]", 1))

	apiResp, err := makeAPIRequest(ctx, url)
	if err != nil {
		log.Printf("ERROR: API request failed: %v", err)
		http.Error(w, "Failed to query NIF API", http.StatusInternalServerError)
		return
	}

	// Process records
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

	// Handle name-based queries with best match selection
	if isNameQuery {
		best, ok := pickBestMatch(results, payload.ClientName)
		if !ok {
			log.Printf("INFO: No suitable match found for name: %s", payload.ClientName)
			http.Error(w, "No suitable match found", http.StatusNotFound)
			return
		}

		log.Printf("INFO: Best match found - NIF: %d, Title: %s", best.Nif, best.Title)
		log.Printf("INFO: Waiting %v due to API rate limiting before fetching full details", APIRateLimit)

		// Respect API rate limiting policy
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

	// Return results for NIF-based queries
	log.Printf("INFO: Returning %d records for NIF query", len(results))
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(results); err != nil {
		log.Printf("ERROR: Failed to encode response: %v", err)
		http.Error(w, "Failed to encode response", http.StatusInternalServerError)
	}
}
