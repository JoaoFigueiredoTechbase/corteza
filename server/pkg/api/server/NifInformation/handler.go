package nifinformation

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
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
	if len(payload.ClientNif) == 9 {
		query = payload.ClientNif
	} else if payload.ClientName != "" {
		query = payload.ClientName
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
		results = append(results, record)
	}

	// return as JSON (array for consistency)
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(results); err != nil {
		log.Printf("ERROR: failed to encode response: %v\n", err)
		http.Error(w, "failed to encode response", http.StatusInternalServerError)
		return
	}
}
