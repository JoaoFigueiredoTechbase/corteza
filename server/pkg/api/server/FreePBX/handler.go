package freepbx

import (
	"encoding/json"
	"io"
	"log"
	"net/http"
)

func HandleCalculatePrice(w http.ResponseWriter, r *http.Request) {
	body, err := io.ReadAll(r.Body)
	if err != nil {
		log.Printf("ERROR: Failed to read request body: %v", err)
		http.Error(w, "Failed to read request body", http.StatusBadRequest)
		return
	}
	defer r.Body.Close()

	log.Printf("INFO: Received request body length: %d bytes", len(body))
	log.Printf("DEBUG: Raw request body: %s", string(body))

	payload, err := ParseData(body)
	if err != nil {
		log.Printf("ERROR: Failed to decode request body: %v", err)
		http.Error(w, "Invalid JSON payload", http.StatusBadRequest)
		return
	}

	log.Printf("INFO: Parsed payload - Calls: %d, Clients: %d, Prices: %d",
		len(payload.Calls), len(payload.Clients), len(payload.Prices))

	// Parse the JSON strings into proper structs
	calls, err := ParseCallsFromJSON(payload.Calls)
	if err != nil {
		log.Printf("ERROR: Failed to parse calls: %v", err)
		http.Error(w, "Invalid calls data", http.StatusBadRequest)
		return
	}
	log.Printf("INFO: Parsed %d calls successfully", len(calls))

	clients, err := ParseClientsFromJSON(payload.Clients)
	if err != nil {
		log.Printf("ERROR: Failed to parse clients: %v", err)
		http.Error(w, "Invalid clients data", http.StatusBadRequest)
		return
	}
	log.Printf("INFO: Parsed %d clients successfully", len(clients))

	prices, err := ParsePricesFromJSON(payload.Prices)
	if err != nil {
		log.Printf("ERROR: Failed to parse prices: %v", err)
		http.Error(w, "Invalid prices data", http.StatusBadRequest)
		return
	}
	log.Printf("INFO: Parsed %d prices successfully", len(prices))

	priceMap := BuildPriceMap(prices)
	log.Printf("INFO: Built price map with %d entries", len(priceMap))

	clientMap := BuildClientMap(clients)
	log.Printf("INFO: Built client map with %d entries", len(clientMap))

	responses := BuildFullPriceResponses(calls, priceMap, clientMap)
	log.Printf("INFO: Generated %d calls and %d clients in response",
		len(responses.Calls), len(responses.Clients))

	if len(responses.Calls) == 0 {
		log.Printf("WARNING: No responses generated - checking data...")

		// Debug: Log first few calls and prices for inspection
		if len(calls) > 0 {
			log.Printf("DEBUG: First call - Dst: %s, BillSec: %s, Sequence: %s, TrunkClientRecord: %s",
				calls[0].Value.Dst, calls[0].Value.BillSec, calls[0].Value.Sequence, calls[0].Value.TrunkClientRecord)
		}

		if len(clients) > 0 {
			log.Printf("DEBUG: First client - Record: %s, RecordID: %s",
				clients[0].Value.ClientRecord, clients[0].Value.RecordID)
		}

		if len(prices) > 0 {
			log.Printf("DEBUG: First price - CountryCode: %s, Type: %s, Price: %s",
				prices[0].Value.CountryCode, prices[0].Value.Type, prices[0].Value.Price)
		}

		// Test number parsing for first call if available
		if len(calls) > 0 {
			region, isMobile, err := GetNumberInfo(calls[0].Value.Dst)
			if err != nil {
				log.Printf("DEBUG: Failed to parse number %s: %v", calls[0].Value.Dst, err)
			} else {
				log.Printf("DEBUG: Number %s - Region: %s, Mobile: %t", calls[0].Value.Dst, region, isMobile)

				// Check if price exists for this region/type
				typeKey := "other"
				if isMobile {
					typeKey = "mobile"
				}
				priceKey := region + "_" + typeKey
				if price, exists := priceMap[priceKey]; exists {
					log.Printf("DEBUG: Found price for key %s: %+v", priceKey, price)
				} else {
					log.Printf("DEBUG: No price found for key %s", priceKey)
					log.Printf("DEBUG: Available price keys (first 10):")
					count := 0
					for key := range priceMap {
						if count >= 10 {
							break
						}
						log.Printf("DEBUG: - %s", key)
						count++
					}
				}
			}
		}
	}

	for _, r := range responses.Calls {
		log.Printf("RESULT: Seq %s: %s call costs %.2f (country %s)",
			r.Sequence, r.CallType, r.CallPrice, r.CountryName)
	}

	for _, c := range responses.Clients {
		log.Printf("SUMMARY: Client %s -> total %.2f (national %.2f, international %.2f)",
			c.ClientRecord, c.TotalCost, c.NationalCost, c.InternationalCost)
	}

	// Set response header and return the responses
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(responses); err != nil {
		log.Printf("ERROR: Failed to encode response: %v", err)
		http.Error(w, "Failed to encode response", http.StatusInternalServerError)
		return
	}

	log.Printf("INFO: Response sent successfully with %d items", len(responses.Calls))
}

// Request structure for the phone number test
type PhoneNumberTestRequest struct {
	Number string `json:"number"`
}

// Response structure for the phone number test
type PhoneNumberTestResponse struct {
	Number   string `json:"number"`
	Region   string `json:"region"`
	IsMobile bool   `json:"is_mobile"`
	Valid    bool   `json:"valid"`
	Type     string `json:"type"`
	Error    string `json:"error,omitempty"`
}

// HandlePhoneNumberTest tests phone number parsing and returns detailed info
func HandlePhoneNumberTest(w http.ResponseWriter, r *http.Request) {
	// Set response headers
	w.Header().Set("Content-Type", "application/json")

	// Read request body
	body, err := io.ReadAll(r.Body)
	if err != nil {
		log.Printf("ERROR: Failed to read request body: %v", err)
		http.Error(w, "Failed to read request body", http.StatusBadRequest)
		return
	}
	defer r.Body.Close()

	// Parse JSON request
	var request PhoneNumberTestRequest
	if err := json.Unmarshal(body, &request); err != nil {
		log.Printf("ERROR: Failed to decode request body: %v", err)
		http.Error(w, "Invalid JSON payload", http.StatusBadRequest)
		return
	}

	// Validate that number is provided
	if request.Number == "" {
		response := PhoneNumberTestResponse{
			Error: "Phone number is required",
		}
		json.NewEncoder(w).Encode(response)
		return
	}

	// Test the phone number
	response := testPhoneNumber(request.Number)

	// Send response
	if err := json.NewEncoder(w).Encode(response); err != nil {
		log.Printf("ERROR: Failed to encode response: %v", err)
		http.Error(w, "Failed to encode response", http.StatusInternalServerError)
		return
	}
}
