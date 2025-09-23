package freepbx

import (
	"encoding/json"
	"fmt"
	"log"
	"math"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/nyaruka/phonenumbers"
)

func parseCallDate(dateStr string) time.Time {
	// Remove timezone info if present for consistent parsing
	if strings.Contains(dateStr, " +") {
		parts := strings.Split(dateStr, " +")
		dateStr = parts[0]
	}

	// Try different date formats
	formats := []string{
		"2006-01-02 15:04:05",
		"2006-01-02T15:04:05",
		"02/01/2006 15:04:05",
		time.RFC3339,
	}

	for _, format := range formats {
		if t, err := time.Parse(format, dateStr); err == nil {
			return t
		}
	}

	// Return zero time if parsing fails
	return time.Time{}
}

func ParseData(data []byte) (*HandleCalculatePriceBody, error) {
	var root HandleCalculatePriceBody
	err := json.Unmarshal(data, &root)
	if err != nil {
		return nil, err
	}
	return &root, nil
}

func ParseCallsFromJSON(callsJSON []string) ([]KV[CallValue], error) {
	var calls []KV[CallValue]

	for _, callStr := range callsJSON {
		var callArray []KV[CallValue]
		if err := json.Unmarshal([]byte(callStr), &callArray); err != nil {
			return nil, fmt.Errorf("failed to parse calls JSON: %v", err)
		}
		calls = append(calls, callArray...)
	}

	return calls, nil
}

func ParseClientsFromJSON(clientsJSON []string) ([]KV[ClientValue], error) {
	var clients []KV[ClientValue]

	for _, clientStr := range clientsJSON {
		var rawClients []struct {
			Value struct {
				ClientRecord  string `json:"client_record"`
				PlanCountries string `json:"plan_countries"` // JSON string
				RecordID      string `json:"recordID"`
				ServiceTime   string `json:"service_time"`
			} `json:"@value"`
			Type string `json:"@type"`
		}

		if err := json.Unmarshal([]byte(clientStr), &rawClients); err != nil {
			return nil, fmt.Errorf("failed to parse clients JSON: %v", err)
		}

		for _, rc := range rawClients {
			var countries []string

			// Handle empty or invalid plan_countries
			if rc.Value.PlanCountries != "" && rc.Value.PlanCountries != "null" {
				// Parse the inner JSON string for countries
				var countryKVs []struct {
					Value string `json:"@value"`
					Type  string `json:"@type"`
				}

				if err := json.Unmarshal([]byte(rc.Value.PlanCountries), &countryKVs); err != nil {
					log.Printf("WARNING: Failed to parse plan_countries for client %s, using empty list: %v", rc.Value.ClientRecord, err)
					// Continue with empty countries list instead of returning error
					countries = []string{}
				} else {
					// Remove duplicates from country codes
					countryMap := make(map[string]bool)
					for _, c := range countryKVs {
						upperCountry := strings.ToUpper(c.Value)
						if !countryMap[upperCountry] {
							countryMap[upperCountry] = true
							countries = append(countries, upperCountry)
						}
					}
				}
			}

			log.Printf("DEBUG: Client %s has %d plan countries: %v", rc.Value.ClientRecord, len(countries), countries)

			clients = append(clients, KV[ClientValue]{
				Value: ClientValue{
					ClientRecord:  rc.Value.ClientRecord,
					PlanCountries: countries,
					RecordID:      rc.Value.RecordID,
					ServiceTime:   rc.Value.ServiceTime,
				},
				Type: rc.Type,
			})
		}
	}

	return clients, nil
}

func ParsePricesFromJSON(pricesJSON []string) ([]KV[PriceValue], error) {
	var prices []KV[PriceValue]

	for _, priceStr := range pricesJSON {
		var priceArray []KV[PriceValue]
		if err := json.Unmarshal([]byte(priceStr), &priceArray); err != nil {
			return nil, fmt.Errorf("failed to parse prices JSON: %v", err)
		}
		prices = append(prices, priceArray...)
	}

	return prices, nil
}

func BuildPriceMap(prices []KV[PriceValue]) map[string]PriceValue {
	priceMap := make(map[string]PriceValue)
	for _, kv := range prices {
		key := kv.Value.CountryCode + "_" + kv.Value.Type
		priceMap[key] = kv.Value
	}
	return priceMap
}

func GetNumberInfo(number string) (region string, isMobile bool, err error) {
	var parsed *phonenumbers.PhoneNumber
	if strings.HasPrefix(number, "+") {
		parsed, err = phonenumbers.Parse(number, "")
	} else {
		parsed, err = phonenumbers.Parse(number, "PT") // fallback region
	}
	if err != nil {
		return "", false, err
	}

	region = phonenumbers.GetRegionCodeForNumber(parsed)
	numType := phonenumbers.GetNumberType(parsed)
	isMobile = numType == phonenumbers.MOBILE || numType == phonenumbers.FIXED_LINE_OR_MOBILE

	return region, isMobile, nil
}

func GetCallPrice(priceMap map[string]PriceValue, region string, isMobile bool) (PriceValue, bool) {
	typeKey := "other"
	if isMobile {
		typeKey = "mobile"
	}
	key := region + "_" + typeKey

	price, ok := priceMap[key]
	if !ok {
		return PriceValue{}, false
	}
	return price, true
}

func CalculateBillableSeconds(billSec int, callRating string) (int, error) {
	parts := strings.Split(callRating, "/")
	if len(parts) != 2 {
		return 0, fmt.Errorf("invalid call rating format: %s", callRating)
	}

	firstSec, err := strconv.Atoi(parts[0])
	if err != nil {
		return 0, err
	}

	subsequentSec, err := strconv.Atoi(parts[1])
	if err != nil {
		return 0, err
	}

	if billSec <= firstSec {
		return firstSec, nil
	}

	extra := billSec - firstSec
	increments := int(math.Ceil(float64(extra) / float64(subsequentSec)))
	total := firstSec + increments*subsequentSec
	return total, nil
}

func CalculateCallPrice(billSec int, callRating string, pricePerMinute float64) (float64, error) {
	billableSec, err := CalculateBillableSeconds(billSec, callRating)
	if err != nil {
		return 0, err
	}
	return (float64(billableSec) / 60.0) * pricePerMinute, nil
}

func BuildClientMap(clients []KV[ClientValue]) map[string]ClientValue {
	clientMap := make(map[string]ClientValue)
	for _, kv := range clients {
		clientMap[kv.Value.RecordID] = kv.Value
	}
	return clientMap
}

// testPhoneNumber processes a phone number and returns detailed information
func testPhoneNumber(number string) PhoneNumberTestResponse {
	response := PhoneNumberTestResponse{
		Number: number,
	}

	// Parse the phone number
	parsed, err := phonenumbers.Parse(number, "")
	if err != nil {
		response.Error = err.Error()
		response.Valid = false
		return response
	}

	// Check if it's valid
	response.Valid = phonenumbers.IsValidNumber(parsed)

	// Get region code
	response.Region = phonenumbers.GetRegionCodeForNumber(parsed)

	// Get number type
	numType := phonenumbers.GetNumberType(parsed)
	response.Type = getNumberTypeString(numType)
	response.IsMobile = numType == phonenumbers.MOBILE || numType == phonenumbers.FIXED_LINE_OR_MOBILE

	return response
}

// getNumberTypeString converts phonenumbers.PhoneNumberType to string
func getNumberTypeString(numType phonenumbers.PhoneNumberType) string {
	switch numType {
	case phonenumbers.FIXED_LINE:
		return "fixed_line"
	case phonenumbers.MOBILE:
		return "mobile"
	case phonenumbers.FIXED_LINE_OR_MOBILE:
		return "fixed_line_or_mobile"
	case phonenumbers.TOLL_FREE:
		return "toll_free"
	case phonenumbers.PREMIUM_RATE:
		return "premium_rate"
	case phonenumbers.SHARED_COST:
		return "shared_cost"
	case phonenumbers.VOIP:
		return "voip"
	case phonenumbers.PERSONAL_NUMBER:
		return "personal_number"
	case phonenumbers.PAGER:
		return "pager"
	case phonenumbers.UAN:
		return "uan"
	case phonenumbers.VOICEMAIL:
		return "voicemail"
	case phonenumbers.UNKNOWN:
		return "unknown"
	default:
		return "unknown"
	}
}

func isCountryInPlan(region string, planCountries []string) bool {
	upperRegion := strings.ToUpper(region)
	for _, country := range planCountries {
		if strings.ToUpper(country) == upperRegion {
			return true
		}
	}
	return false
}

func BuildFullPriceResponses(calls []KV[CallValue], priceMap map[string]PriceValue, clientMap map[string]ClientValue) CalculatePriceFullResponse {
	// Sort calls by date (chronological order)
	sort.Slice(calls, func(i, j int) bool {
		dateI := parseCallDate(calls[i].Value.Calldate)
		dateJ := parseCallDate(calls[j].Value.Calldate)
		return dateI.Before(dateJ)
	})

	callDetails := make([]CallDetail, 0, len(calls))
	clientTotals := make(map[string]*ClientSummary)

	for _, call := range calls {
		dst := call.Value.Dst
		billSec, err := strconv.Atoi(call.Value.BillSec)
		if err != nil {
			log.Printf("DEBUG: Invalid billsec for call %s: %v", call.Value.Sequence, err)
			continue
		}

		client, ok := clientMap[call.Value.TrunkClientRecord]
		if !ok {
			log.Printf("DEBUG: Client not found for record %s", call.Value.TrunkClientRecord)
			continue
		}

		region, isMobile, err := GetNumberInfo(dst)
		if err != nil {
			log.Printf("DEBUG: Failed to parse number %s: %v", dst, err)
			continue
		}

		priceEntry, ok := GetCallPrice(priceMap, region, isMobile)
		if !ok {
			log.Printf("DEBUG: No price found for region %s, mobile=%t", region, isMobile)
			continue
		}

		countryName := priceEntry.CountryName
		pricePerMinute, _ := strconv.ParseFloat(priceEntry.Price, 64)

		// Check if country is in client's plan (using deduplicated list)
		inPlan := isCountryInPlan(region, client.PlanCountries)

		// Determine call type (national/international)
		isNational := strings.ToUpper(region) == "PT"

		// Initialize client totals if not exists
		totals, exists := clientTotals[client.RecordID]
		if !exists {
			serviceTime, _ := strconv.Atoi(client.ServiceTime)
			totals = &ClientSummary{
				ClientRecord:     client.ClientRecord,
				RecordID:         client.RecordID,
				TotalServiceTime: serviceTime, // Total plan time available
				RemainingTime:    serviceTime, // Time remaining in plan
				UsedPlanTime:     0,           // Time used from plan
				PlanEndDate:      "",          // Date when plan ended
			}
			clientTotals[client.RecordID] = totals
		}

		var callPrice float64
		callType := "other"
		if isMobile {
			callType = "mobile"
		}

		if inPlan {
			totals.PlanCalls++
			totals.PlanTotalTime += billSec

			if totals.RemainingTime > 0 {
				// Call is within plan time
				if billSec <= totals.RemainingTime {
					// Call completely covered by plan
					totals.UsedPlanTime += billSec
					totals.RemainingTime -= billSec
					callPrice = 0
				} else {
					// Call exceeds remaining plan time
					coveredTime := totals.RemainingTime
					exceededTime := billSec - totals.RemainingTime

					totals.UsedPlanTime += coveredTime
					totals.ExceededPlanTime += exceededTime
					totals.RemainingTime = 0

					// Calculate price only for exceeded time
					exceededPrice, err := CalculateCallPrice(exceededTime, priceEntry.CallRating, pricePerMinute)
					if err != nil {
						log.Printf("DEBUG: Price calc failed for exceeded time: %v", err)
						exceededPrice = 0
					}

					callPrice = exceededPrice
					totals.ExceededPlanCost += exceededPrice

					// Save the date when plan ended
					if totals.PlanEndDate == "" {
						totals.PlanEndDate = call.Value.Calldate
					}
				}
			} else {
				// Plan already ended, charge full call
				totals.ExceededPlanTime += billSec
				exceededPrice, err := CalculateCallPrice(billSec, priceEntry.CallRating, pricePerMinute)
				if err != nil {
					log.Printf("DEBUG: Price calc failed: %v", err)
					exceededPrice = 0
				}
				callPrice = exceededPrice
				totals.ExceededPlanCost += exceededPrice
			}
		} else {
			// Call outside plan
			totals.NonPlanCalls++
			totals.NonPlanTotalTime += billSec

			exceededPrice, err := CalculateCallPrice(billSec, priceEntry.CallRating, pricePerMinute)
			if err != nil {
				log.Printf("DEBUG: Price calc failed: %v", err)
				exceededPrice = 0
			}
			callPrice = exceededPrice
		}

		// Update totals based on call type (national/international)
		if isNational {
			totals.NationalCalls++
			totals.NationalTime += billSec
			totals.NationalCost += math.Round(callPrice*100) / 100
		} else {
			totals.InternationalCalls++
			totals.InternationalTime += billSec
			totals.InternationalCost += math.Round(callPrice*100) / 100
		}

		// Add call details
		callDetails = append(callDetails, CallDetail{
			Sequence:    call.Value.Sequence,
			CdrId:       call.Value.CdrId,
			UniqueId:    call.Value.UniqueId,
			CallPrice:   math.Round(callPrice*100) / 100,
			CountryName: countryName,
			CountryCode: region,
			CallType:    callType,
			InPlan:      inPlan,
			IsNational:  isNational,
		})

		// Update total costs and time
		totals.TotalCost += math.Round(callPrice*100) / 100
		totals.TotalTime += billSec
	}

	// Convert map to slice
	clients := make([]ClientSummary, 0, len(clientTotals))
	for _, c := range clientTotals {
		clients = append(clients, *c)
	}

	return CalculatePriceFullResponse{
		Clients: clients,
		Calls:   callDetails,
	}
}
