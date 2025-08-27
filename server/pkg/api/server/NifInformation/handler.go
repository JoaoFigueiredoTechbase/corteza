package nifinformation

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"
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

func parseCandidates(respBody []byte) ([]NifApiResponse, error) {
	var searchResp SearchResponse
	if err := json.Unmarshal(respBody, &searchResp); err != nil {
		return nil, err
	}

	var candidates []NifApiResponse
	for _, raw := range searchResp.Records {
		var lw LightweightRecord
		if err := json.Unmarshal(raw, &lw); err != nil {
			continue // skip bad entries
		}
		if lw.Nif == 0 {
			continue // skip ones without nif
		}

		candidates = append(candidates, NifApiResponse{
			Nif:        lw.Nif,
			Title:      lw.Title,
			City:       lw.City,
			Pc4:        lw.Pc4,
			Pc3:        lw.Pc3,
			RaciusLink: lw.Racius,
		})
	}
	return candidates, nil
}

func pickBestMatch(records []NifApiResponse, query string) (NifApiResponse, bool) {
	query = strings.ToLower(query)
	queryWords := strings.Fields(query)

	bestScore := -1
	var best NifApiResponse
	found := false

	for _, rec := range records {
		title := strings.ToLower(rec.Title)

		score := 0

		// exact substring
		if strings.Contains(title, query) {
			score += 5
		}

		// starts with
		if strings.HasPrefix(title, query) {
			score += 3
		}

		// word-by-word check
		lastIndex := -1
		inOrder := true
		for _, w := range queryWords {
			idx := strings.Index(title, w)
			if idx == -1 {
				inOrder = false
				continue
			}
			score++
			if lastIndex != -1 && idx < lastIndex {
				inOrder = false
			}
			lastIndex = idx
		}
		if inOrder {
			score += 2
		}

		if score > bestScore {
			bestScore = score
			best = rec
			found = true
		}
	}

	return best, found
}

// helper to map one record
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
		return NifApiResponse{}, fmt.Errorf("unmarshal record: %w", err)
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

func fetchFullInfo(nif int, apiKey string) (NifApiResponse, error) {
	url := fmt.Sprintf("http://www.nif.pt/?json=1&q=%d&key=%s", nif, apiKey)
	resp, err := http.Get(url)
	if err != nil {
		return NifApiResponse{}, err
	}
	defer resp.Body.Close()

	var apiResp ApiResponse
	if err := json.NewDecoder(resp.Body).Decode(&apiResp); err != nil {
		return NifApiResponse{}, err
	}

	// assuming only 1 record for a NIF
	for _, raw := range apiResp.Records {
		return parseRecord(raw)
	}

	return NifApiResponse{}, fmt.Errorf("no record found for nif %d", nif)
}

func HandleClientInformationSearch(w http.ResponseWriter, r *http.Request) {
	log.Println("Received Client Information Search Request")

	var payload ClientInformation
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		log.Printf("ERROR: Failed to decode request body: %v\n", err)
		http.Error(w, "invalid request", http.StatusBadRequest)
		return
	}

	if len(payload.ApiKey) == 0 {
		http.Error(w, "Missing api key", http.StatusBadRequest)
		return
	}

	// build query param
	var query string
	isNameQuery := false
	if len(payload.ClientNif) == 9 {
		query = payload.ClientNif
	} else if payload.ClientName != "" {
		query = payload.ClientName
		isNameQuery = true
	} else {
		http.Error(w, "either client_nif (9 digits) or client_name required", http.StatusBadRequest)
		return
	}

	url := fmt.Sprintf("http://www.nif.pt/?json=1&q=%s&key=%s", query, payload.ApiKey)
	resp, err := http.Get(url)
	if err != nil {
		log.Printf("ERROR: Failed to make request %s: %v\n", url, err)
		http.Error(w, "failed to make request", http.StatusInternalServerError)
		return
	}
	defer resp.Body.Close()

	var apiResp ApiResponse
	if err := json.NewDecoder(resp.Body).Decode(&apiResp); err != nil {
		log.Printf("ERROR: Failed to decode JSON: %v\n", err)
		http.Error(w, "failed to decode response", http.StatusInternalServerError)
		return
	}

	// collect records
	var results []NifApiResponse
	for _, raw := range apiResp.Records {
		record, err := parseRecord(raw)
		if err != nil {
			log.Printf("ERROR: %v\n", err)
			continue
		}
		if record.Nif != 0 {
			results = append(results, record)
		}
	}
	if isNameQuery {
		if best, ok := pickBestMatch(results, payload.ClientName); ok {
			// wait 1 minute before making the NIF request
			time.Sleep(time.Minute)

			fullRecord, err := fetchFullInfo(best.Nif, payload.ApiKey)
			if err != nil {
				log.Printf("ERROR: Failed to fetch full info for NIF %d: %v", best.Nif, err)
				http.Error(w, "failed to fetch full info", http.StatusInternalServerError)
				return
			}

			w.Header().Set("Content-Type", "application/json")
			if err := json.NewEncoder(w).Encode(fullRecord); err != nil {
				log.Printf("ERROR: failed to encode full record: %v", err)
				http.Error(w, "failed to encode response", http.StatusInternalServerError)
			}
			return
		}

		http.Error(w, "no valid match found", http.StatusNotFound)
		return
	}

	// return as JSON (array for consistency)
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(results); err != nil {
		log.Printf("ERROR: failed to encode response: %v\n", err)
		http.Error(w, "failed to encode response", http.StatusInternalServerError)
		return
	}
}
