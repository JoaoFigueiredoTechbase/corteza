package nifinformation

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"regexp"
	"strings"
	"time"
)

func validateNif(nif string) bool {
	if len(nif) != 9 {
		return false
	}
	matched, _ := regexp.MatchString(`^\d{9}$`, nif)
	return matched
}

func sanitizeQuery(query string) string {
	query = strings.TrimSpace(query)
	reg := regexp.MustCompile(`[^a-zA-Z0-9\s\-.,&()]`)
	return reg.ReplaceAllString(query, "")
}

func createHTTPClient() *http.Client {
	return &http.Client{
		Timeout: RequestTimeout,
	}
}

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

		if title == query {
			return rec, true
		}

		if strings.Contains(title, query) {
			score += 10
		}

		if strings.HasPrefix(title, query) {
			score += 7
		}

		if strings.HasSuffix(title, query) {
			score += 5
		}

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

		score += wordMatches * 2
		if inOrder && wordMatches > 1 {
			score += 3
		}

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

func parseRecord(raw json.RawMessage) (NifApiResponse, error) {
	var tmp struct {
		Nif      int             `json:"nif"`
		Title    string          `json:"title"`
		Address  string          `json:"address"`
		Pc4      string          `json:"pc4"`
		Pc3      string          `json:"pc3"`
		City     string          `json:"city"`
		Activity string          `json:"activity"`
		Status   string          `json:"status"`
		CaeRaw   json.RawMessage `json:"cae"`
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

	var caeList []string
	if len(tmp.CaeRaw) > 0 {
		// try as array
		if err := json.Unmarshal(tmp.CaeRaw, &caeList); err != nil {
			// try as string
			var single string
			if err := json.Unmarshal(tmp.CaeRaw, &single); err == nil {
				caeList = []string{single}
			} else {
				return NifApiResponse{}, fmt.Errorf("invalid cae format: %s", string(tmp.CaeRaw))
			}
		}
	}

	return NifApiResponse{
		Nif:        tmp.Nif,
		Title:      tmp.Title,
		Address:    tmp.Address,
		Pc4:        tmp.Pc4,
		Pc3:        tmp.Pc3,
		City:       tmp.City,
		Activity:   tmp.Activity,
		Status:     tmp.Status,
		CaeList:    caeList,
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

	for _, raw := range apiResp.Records {
		return parseRecord(raw)
	}

	return NifApiResponse{}, fmt.Errorf("no record found for NIF %d", nif)
}

const usageFile = "usage.txt"

func loadUsage() (Usage, error) {
	var u Usage
	data, err := os.ReadFile(usageFile)
	if err != nil {
		if os.IsNotExist(err) {
			return Usage{}, nil
		}
		return u, err
	}
	err = json.Unmarshal(data, &u)
	return u, err
}

func saveUsage(u Usage) error {
	data, err := json.Marshal(u)
	if err != nil {
		return err
	}
	return os.WriteFile(usageFile, data, 0644)
}

func checkAndUpdateQuota(u *Usage, limits RateLimits) bool {
	now := time.Now()

	// Reset counters if time passed
	if now.Month() != u.LastUpdate.Month() {
		u.Month = 0
	}
	if now.YearDay() != u.LastUpdate.YearDay() {
		u.Day = 0
	}
	if now.Hour() != u.LastUpdate.Hour() {
		u.Hour = 0
	}
	if now.Minute() != u.LastUpdate.Minute() {
		u.Minute = 0
	}

	// Check limits
	if u.Month >= limits.Month || u.Day >= limits.Day || u.Hour >= limits.Hour || u.Minute >= limits.Minute {
		return false
	}

	// Increment counters
	u.Month++
	u.Day++
	u.Hour++
	u.Minute++
	u.LastUpdate = now
	return true
}
