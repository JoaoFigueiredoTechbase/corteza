package freepbx

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
)

func HandleCalculatePrice(w http.ResponseWriter, r *http.Request) {
	body, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "Failed to read request body", http.StatusBadRequest)
		return
	}
	defer r.Body.Close()

	payload, err := ParseData(body)
	if err != nil {
		log.Printf("ERROR: Failed to decode request body: %v", err)
		http.Error(w, "Invalid JSON payload", http.StatusBadRequest)
		return
	}

	priceMap := BuildPriceMap(payload.Prices)
	responses := BuildCalculatePriceResponses(payload.Calls, priceMap)

	for _, r := range responses {
		fmt.Printf("Seq %s: %s call costs %.2f (price record %s)\n", r.Sequence, r.CallType, r.CallPrice, r.PriceRecord)
	}

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
