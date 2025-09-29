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

		if len(calls) > 0 {
			log.Printf("DEBUG: First call - Dst: %s, BillSec: %s, Sequence: %s, TrunkClientRecord: %s",
				calls[0].Value.Dst, calls[0].Value.BillSec, calls[0].Value.Sequence, calls[0].Value.TrunkClientRecord)
		}

		if len(calls) > 0 {
			region, isMobile, ptCallType, err := GetNumberInfo(calls[0].Value.Dst)
			if err != nil {
				log.Printf("DEBUG: Failed to parse number %s: %v", calls[0].Value.Dst, err)
			} else {
				log.Printf("DEBUG: Number %s - Region: %s, Mobile: %t", calls[0].Value.Dst, region, isMobile)

				if ptCallType != nil {
					log.Printf("DEBUG: Portuguese call type - Type: %s, Description: %s, Prefix: %s, Category: %s",
						ptCallType.Type, ptCallType.Description, ptCallType.Prefix, ptCallType.Category)
				}

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

	// Log individual call details with classifications
	for _, r := range responses.Calls {
		logMsg := fmt.Sprintf("RESULT: Seq %s: %s call costs %.2f (country %s)",
			r.Sequence, r.CallType, r.CallPrice, r.CountryName)

		if r.PortugueseCallType != nil {
			logMsg += fmt.Sprintf(" [PT: %s - %s]", r.PortugueseCallType.Type, r.PortugueseCallType.Description)
		}

		if r.CallerType != nil {
			logMsg += fmt.Sprintf(" [Caller: %s/%s]", r.CallerType.Type, r.CallerType.Classification)
		}

		if r.DestinationType != nil {
			logMsg += fmt.Sprintf(" [Dest: %s]", r.DestinationType.Type)
		}

		log.Print(logMsg)
	}

	// Log client summaries
	for _, c := range responses.Clients {
		log.Printf("SUMMARY: Client %s -> total %.2f (national %.2f, international %.2f)",
			c.ClientRecord, c.TotalCost, c.NationalCost, c.InternationalCost)

		log.Printf("PORTUGUESE STATS: Client %s -> Mobile: %d calls/%.2f€, Landline: %d calls/%.2f€, Premium: %d calls/%.2f€, Free: %d calls/%.2f€",
			c.ClientRecord,
			c.MobileCalls, c.MobileCost,
			c.LandlineCalls, c.LandlineCost,
			c.PremiumCalls, c.PremiumCost,
			c.FreeCalls, c.FreeCost)

		if c.SharedCostCalls > 0 || c.InternetCalls > 0 || c.AudiotextCalls > 0 || c.SpecialServiceCalls > 0 {
			log.Printf("PORTUGUESE SPECIAL: Client %s -> SharedCost: %d/%.2f€, Internet: %d/%.2f€, Audiotext: %d/%.2f€, Special: %d/%.2f€",
				c.ClientRecord,
				c.SharedCostCalls, c.SharedCostCost,
				c.InternetCalls, c.InternetCost,
				c.AudiotextCalls, c.AudiotextCost,
				c.SpecialServiceCalls, c.SpecialServiceCost)
		}

		// Log geographic caller statistics
		if len(c.GeographicCallers) > 0 {
			log.Printf("GEOGRAPHIC CALLERS: Client %s has %d geographic (landline) callers", c.ClientRecord, len(c.GeographicCallers))
			for callerNum, stats := range c.GeographicCallers {
				log.Printf("  📞 Caller %d: Total=%d calls/%d min",
					callerNum, stats.TotalCalls, stats.TotalMinutes)

				log.Printf("    └─ Landline: %d calls/%d min | Mobile: %d calls/%d min | International: %d calls/%d min",
					stats.LandlineCalls, stats.LandlineMinutes,
					stats.MobileCalls, stats.MobileMinutes,
					stats.InternationalCalls, stats.InternationalMinutes)

				if stats.NonGeographicCalls > 0 || stats.ShortCalls > 0 || stats.ValueAddedCalls > 0 || stats.NomadCalls > 0 {
					log.Printf("    └─ NonGeo: %d/%d min | Short: %d/%d min | Nomad: %d/%d min",
						stats.NonGeographicCalls, stats.NonGeographicMinutes,
						stats.ShortCalls, stats.ShortMinutes,
						stats.NomadCalls, stats.NomadMinutes)
				}

				if stats.ValueAddedCalls > 0 {
					log.Printf("    └─ ValueAdded (760/761): %d calls/%d min | Specifically 760: %d calls/%d min",
						stats.ValueAddedCalls, stats.ValueAddedMinutes,
						stats.Value760Calls, stats.Value760Minutes)
				}
			}
		}

		// Log nomad caller statistics
		if len(c.NomadCallers) > 0 {
			log.Printf("NOMAD CALLERS: Client %s has %d nomad callers", c.ClientRecord, len(c.NomadCallers))
			for callerNum, stats := range c.NomadCallers {
				log.Printf("  📱 Caller %d: Total=%d calls/%d min | International=%d calls/%d min",
					callerNum,
					stats.TotalCalls, stats.TotalMinutes,
					stats.InternationalCalls, stats.InternationalMinutes)
			}
		}
	}

	// Send JSON response
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(responses); err != nil {
		log.Printf("ERROR: Failed to encode response: %v", err)
		http.Error(w, "Failed to encode response", http.StatusInternalServerError)
		return
	}

	log.Printf("INFO: Response sent successfully with %d calls and caller classification data", len(responses.Calls))
}

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
