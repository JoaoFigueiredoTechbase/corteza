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

func GetNumberInfo(number string) (region string, isMobile bool, ptCallType *PortugueseCallType, err error) {
	var parsed *phonenumbers.PhoneNumber
	if strings.HasPrefix(number, "+") {
		parsed, err = phonenumbers.Parse(number, "")
	} else {
		parsed, err = phonenumbers.Parse(number, "PT") // fallback region
	}
	if err != nil {
		return "", false, nil, err
	}

	region = phonenumbers.GetRegionCodeForNumber(parsed)
	numType := phonenumbers.GetNumberType(parsed)
	isMobile = numType == phonenumbers.MOBILE || numType == phonenumbers.FIXED_LINE_OR_MOBILE

	// Get Portuguese-specific details if it's a Portuguese number
	var callType *PortugueseCallType
	if region == "PT" {
		ptType := GetPortugueseCallType(number)
		callType = &ptType
	}

	return region, isMobile, callType, nil
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

// Helper function to format date as YYYY-MM-DD
func formatDateKey(t time.Time) string {
	return t.Format("2006-01-02")
}

// Updated BuildFullPriceResponses function with Portuguese call type counts
// func BuildFullPriceResponses(calls []KV[CallValue], priceMap map[string]PriceValue, clientMap map[string]ClientValue) CalculatePriceFullResponse {
// 	// Sort calls by date (chronological order)
// 	sort.Slice(calls, func(i, j int) bool {
// 		dateI := parseCallDate(calls[i].Value.Calldate)
// 		dateJ := parseCallDate(calls[j].Value.Calldate)
// 		return dateI.Before(dateJ)
// 	})

// 	callDetails := make([]CallDetail, 0, len(calls))
// 	clientTotals := make(map[string]*ClientSummary)

// 	// Track daily statistics for each client
// 	clientDailyStats := make(map[string]map[string]*DailySummary) // client_id -> date -> daily_stats

// 	for _, call := range calls {
// 		dst := call.Value.Dst
// 		billSec, err := strconv.Atoi(call.Value.BillSec)
// 		if err != nil {
// 			log.Printf("DEBUG: Invalid billsec for call %s: %v", call.Value.Sequence, err)
// 			continue
// 		}

// 		client, ok := clientMap[call.Value.TrunkClientRecord]
// 		if !ok {
// 			log.Printf("DEBUG: Client not found for record %s", call.Value.TrunkClientRecord)
// 			continue
// 		}

// 		region, isMobile, ptCallType, err := GetNumberInfo(dst)
// 		if err != nil {
// 			log.Printf("DEBUG: Failed to parse number %s: %v", dst, err)
// 			continue
// 		}

// 		priceEntry, ok := GetCallPrice(priceMap, region, isMobile)
// 		if !ok {
// 			log.Printf("DEBUG: No price found for region %s, mobile=%t", region, isMobile)
// 			continue
// 		}

// 		countryName := priceEntry.CountryName
// 		pricePerMinute, _ := strconv.ParseFloat(priceEntry.Price, 64)

// 		// Check if country is in client's plan
// 		inPlan := isCountryInPlan(region, client.PlanCountries)

// 		// Determine call type (national/international)
// 		isNational := strings.ToUpper(region) == "PT"

// 		// Get call date
// 		callDate := parseCallDate(call.Value.Calldate)
// 		dateKey := formatDateKey(callDate)

// 		// Initialize client totals if not exists
// 		totals, exists := clientTotals[client.RecordID]
// 		if !exists {
// 			serviceTime, _ := strconv.Atoi(client.ServiceTime)
// 			totals = &ClientSummary{
// 				ClientRecord:     client.ClientRecord,
// 				RecordID:         client.RecordID,
// 				TotalServiceTime: serviceTime,
// 				RemainingTime:    serviceTime,
// 				UsedPlanTime:     0,
// 				PlanEndDate:      "",
// 			}
// 			clientTotals[client.RecordID] = totals
// 		}

// 		// Initialize daily stats for this client if not exists
// 		if clientDailyStats[client.RecordID] == nil {
// 			clientDailyStats[client.RecordID] = make(map[string]*DailySummary)
// 		}

// 		dailyStats, exists := clientDailyStats[client.RecordID][dateKey]
// 		if !exists {
// 			dailyStats = &DailySummary{
// 				Date:                 dateKey,
// 				RemainingTimeAtStart: totals.RemainingTime,
// 			}
// 			clientDailyStats[client.RecordID][dateKey] = dailyStats
// 		}

// 		var callPrice float64
// 		callType := "other"
// 		if isMobile {
// 			callType = "mobile"
// 		}

// 		// Store remaining time at start of processing this call
// 		remainingAtStart := totals.RemainingTime

// 		if inPlan {
// 			totals.PlanCalls++
// 			totals.PlanTotalTime += billSec
// 			dailyStats.PlanCalls++
// 			dailyStats.PlanTotalTime += billSec

// 			if totals.RemainingTime > 0 {
// 				if billSec <= totals.RemainingTime {
// 					totals.UsedPlanTime += billSec
// 					totals.RemainingTime -= billSec
// 					dailyStats.UsedPlanTime += billSec
// 					callPrice = 0
// 				} else {
// 					coveredTime := totals.RemainingTime
// 					exceededTime := billSec - totals.RemainingTime

// 					totals.UsedPlanTime += coveredTime
// 					totals.ExceededPlanTime += exceededTime
// 					totals.RemainingTime = 0

// 					dailyStats.UsedPlanTime += coveredTime
// 					dailyStats.ExceededPlanTime += exceededTime

// 					exceededPrice, err := CalculateCallPrice(exceededTime, priceEntry.CallRating, pricePerMinute)
// 					if err != nil {
// 						log.Printf("DEBUG: Price calc failed for exceeded time: %v", err)
// 						exceededPrice = 0
// 					}

// 					callPrice = exceededPrice
// 					totals.ExceededPlanCost += exceededPrice
// 					dailyStats.ExceededPlanCost += exceededPrice

// 					if totals.PlanEndDate == "" {
// 						totals.PlanEndDate = call.Value.Calldate
// 						dailyStats.PlanEndedThisDay = true
// 					}
// 				}
// 			} else {
// 				totals.ExceededPlanTime += billSec
// 				dailyStats.ExceededPlanTime += billSec

// 				exceededPrice, err := CalculateCallPrice(billSec, priceEntry.CallRating, pricePerMinute)
// 				if err != nil {
// 					log.Printf("DEBUG: Price calc failed: %v", err)
// 					exceededPrice = 0
// 				}
// 				callPrice = exceededPrice
// 				totals.ExceededPlanCost += exceededPrice
// 				dailyStats.ExceededPlanCost += exceededPrice
// 			}
// 		} else {
// 			totals.NonPlanCalls++
// 			totals.NonPlanTotalTime += billSec
// 			dailyStats.NonPlanCalls++
// 			dailyStats.NonPlanTotalTime += billSec

// 			exceededPrice, err := CalculateCallPrice(billSec, priceEntry.CallRating, pricePerMinute)
// 			if err != nil {
// 				log.Printf("DEBUG: Price calc failed: %v", err)
// 				exceededPrice = 0
// 			}
// 			callPrice = exceededPrice
// 		}

// 		// Update totals and daily stats based on call type (national/international)
// 		roundedPrice := math.Round(callPrice*100) / 100

// 		if isNational {
// 			totals.NationalCalls++
// 			totals.NationalTime += billSec
// 			totals.NationalCost += roundedPrice

// 			dailyStats.NationalCalls++
// 			dailyStats.NationalTime += billSec
// 			dailyStats.NationalCost += roundedPrice
// 		} else {
// 			totals.InternationalCalls++
// 			totals.InternationalTime += billSec
// 			totals.InternationalCost += roundedPrice

// 			dailyStats.InternationalCalls++
// 			dailyStats.InternationalTime += billSec
// 			dailyStats.InternationalCost += roundedPrice
// 		}

// 		// Update Portuguese call counts if it's a Portuguese number
// 		if ptCallType != nil && isNational {
// 			switch ptCallType.Type {
// 			case "landline":
// 				totals.LandlineCalls++
// 				totals.LandlineCost += roundedPrice
// 				dailyStats.LandlineCalls++
// 				dailyStats.LandlineCost += roundedPrice
// 			case "mobile":
// 				totals.MobileCalls++
// 				totals.MobileCost += roundedPrice
// 				dailyStats.MobileCalls++
// 				dailyStats.MobileCost += roundedPrice
// 			case "premium":
// 				totals.PremiumCalls++
// 				totals.PremiumCost += roundedPrice
// 				dailyStats.PremiumCalls++
// 				dailyStats.PremiumCost += roundedPrice
// 			case "free":
// 				totals.FreeCalls++
// 				totals.FreeCost += roundedPrice
// 				dailyStats.FreeCalls++
// 				dailyStats.FreeCost += roundedPrice
// 			case "shared_cost":
// 				totals.SharedCostCalls++
// 				totals.SharedCostCost += roundedPrice
// 				dailyStats.SharedCostCalls++
// 				dailyStats.SharedCostCost += roundedPrice
// 			case "internet":
// 				totals.InternetCalls++
// 				totals.InternetCost += roundedPrice
// 				dailyStats.InternetCalls++
// 				dailyStats.InternetCost += roundedPrice
// 			case "audiotext":
// 				totals.AudiotextCalls++
// 				totals.AudiotextCost += roundedPrice
// 				dailyStats.AudiotextCalls++
// 				dailyStats.AudiotextCost += roundedPrice
// 			case "special_service":
// 				totals.SpecialServiceCalls++
// 				totals.SpecialServiceCost += roundedPrice
// 				dailyStats.SpecialServiceCalls++
// 				dailyStats.SpecialServiceCost += roundedPrice
// 			}
// 		}

// 		// Update remaining time tracking for daily stats
// 		dailyStats.RemainingTimeAtStart = remainingAtStart
// 		dailyStats.RemainingTimeAtEnd = totals.RemainingTime

// 		// Add call details with Portuguese call type information
// 		callDetails = append(callDetails, CallDetail{
// 			Sequence:           call.Value.Sequence,
// 			CdrId:              call.Value.CdrId,
// 			UniqueId:           call.Value.UniqueId,
// 			CallPrice:          roundedPrice,
// 			CountryName:        countryName,
// 			CountryCode:        region,
// 			CallType:           callType,
// 			InPlan:             inPlan,
// 			IsNational:         isNational,
// 			PortugueseCallType: ptCallType,
// 		})

// 		// Update total costs and time
// 		totals.TotalCost += roundedPrice
// 		totals.TotalTime += billSec

// 		dailyStats.TotalCost += roundedPrice
// 		dailyStats.TotalTime += billSec
// 	}

// 	// Convert client map to slice and add daily statistics
// 	clients := make([]ClientSummary, 0, len(clientTotals))
// 	for clientID, totals := range clientTotals {
// 		if dailyStatsMap, exists := clientDailyStats[clientID]; exists {
// 			dailyStats := make([]DailySummary, 0, len(dailyStatsMap))
// 			for _, daily := range dailyStatsMap {
// 				dailyStats = append(dailyStats, *daily)
// 			}

// 			// Sort daily stats by date
// 			sort.Slice(dailyStats, func(i, j int) bool {
// 				return dailyStats[i].Date < dailyStats[j].Date
// 			})

// 			totals.DailyStats = dailyStats
// 		}

// 		clients = append(clients, *totals)
// 	}

// 	return CalculatePriceFullResponse{
// 		Clients: clients,
// 		Calls:   callDetails,
// 	}
// }

// Add this to your existing helpers.go file, replacing the BuildFullPriceResponses function
func BuildFullPriceResponses(calls []KV[CallValue], priceMap map[string]PriceValue, clientMap map[string]ClientValue) CalculatePriceFullResponse {
	// Sort calls by date (chronological order)
	sort.Slice(calls, func(i, j int) bool {
		dateI := parseCallDate(calls[i].Value.Calldate)
		dateJ := parseCallDate(calls[j].Value.Calldate)
		return dateI.Before(dateJ)
	})

	callDetails := make([]CallDetail, 0, len(calls))
	clientTotals := make(map[string]*ClientSummary)

	// Track daily statistics for each client
	clientDailyStats := make(map[string]map[string]*DailySummary) // client_id -> date -> daily_stats

	for _, call := range calls {
		dst := call.Value.Dst
		src := call.Value.Src
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

		region, isMobile, ptCallType, err := GetNumberInfo(dst)
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

		// Check if country is in client's plan
		inPlan := isCountryInPlan(region, client.PlanCountries)

		// Determine call type (national/international)
		isNational := strings.ToUpper(region) == "PT"

		// Get call date
		callDate := parseCallDate(call.Value.Calldate)
		dateKey := formatDateKey(callDate)

		// Initialize client totals if not exists
		totals, exists := clientTotals[client.RecordID]
		if !exists {
			serviceTime, _ := strconv.Atoi(client.ServiceTime)
			totals = &ClientSummary{
				ClientRecord:      client.ClientRecord,
				RecordID:          client.RecordID,
				TotalServiceTime:  serviceTime,
				RemainingTime:     serviceTime,
				UsedPlanTime:      0,
				PlanEndDate:       "",
				GeographicCallers: []*GeographicCallerStats{},
				NomadCallers:      []*NomadCallerStats{},
			}
			clientTotals[client.RecordID] = totals
		}

		// Initialize daily stats for this client if not exists
		if clientDailyStats[client.RecordID] == nil {
			clientDailyStats[client.RecordID] = make(map[string]*DailySummary)
		}

		dailyStats, exists := clientDailyStats[client.RecordID][dateKey]
		if !exists {
			dailyStats = &DailySummary{
				Date:                 dateKey,
				RemainingTimeAtStart: totals.RemainingTime,
			}
			clientDailyStats[client.RecordID][dateKey] = dailyStats
		}

		var callPrice float64
		callType := "other"
		if isMobile {
			callType = "mobile"
		}

		// Classify caller and destination
		callerType := ClassifyCallerType(src)
		destinationType := ClassifyNumber(dst)

		// Store remaining time at start of processing this call
		remainingAtStart := totals.RemainingTime

		if inPlan {
			totals.PlanCalls++
			totals.PlanTotalTime += billSec
			dailyStats.PlanCalls++
			dailyStats.PlanTotalTime += billSec

			if totals.RemainingTime > 0 {
				if billSec <= totals.RemainingTime {
					totals.UsedPlanTime += billSec
					totals.RemainingTime -= billSec
					dailyStats.UsedPlanTime += billSec
					callPrice = 0
				} else {
					coveredTime := totals.RemainingTime
					exceededTime := billSec - totals.RemainingTime

					totals.UsedPlanTime += coveredTime
					totals.ExceededPlanTime += exceededTime
					totals.RemainingTime = 0

					dailyStats.UsedPlanTime += coveredTime
					dailyStats.ExceededPlanTime += exceededTime

					exceededPrice, err := CalculateCallPrice(exceededTime, priceEntry.CallRating, pricePerMinute)
					if err != nil {
						log.Printf("DEBUG: Price calc failed for exceeded time: %v", err)
						exceededPrice = 0
					}

					callPrice = exceededPrice
					totals.ExceededPlanCost += exceededPrice
					dailyStats.ExceededPlanCost += exceededPrice

					if totals.PlanEndDate == "" {
						totals.PlanEndDate = call.Value.Calldate
						totals.PlanEnded = true // Set the new field
						dailyStats.PlanEndedThisDay = true
					}
				}
			} else {
				totals.ExceededPlanTime += billSec
				dailyStats.ExceededPlanTime += billSec

				exceededPrice, err := CalculateCallPrice(billSec, priceEntry.CallRating, pricePerMinute)
				if err != nil {
					log.Printf("DEBUG: Price calc failed: %v", err)
					exceededPrice = 0
				}
				callPrice = exceededPrice
				totals.ExceededPlanCost += exceededPrice
				dailyStats.ExceededPlanCost += exceededPrice
			}
		} else {
			totals.NonPlanCalls++
			totals.NonPlanTotalTime += billSec
			dailyStats.NonPlanCalls++
			dailyStats.NonPlanTotalTime += billSec

			exceededPrice, err := CalculateCallPrice(billSec, priceEntry.CallRating, pricePerMinute)
			if err != nil {
				log.Printf("DEBUG: Price calc failed: %v", err)
				exceededPrice = 0
			}
			callPrice = exceededPrice
		}

		// Update totals and daily stats based on call type (national/international)
		roundedPrice := math.Round(callPrice*100) / 100

		if isNational {
			totals.NationalCalls++
			totals.NationalTime += billSec
			totals.NationalCost += roundedPrice

			dailyStats.NationalCalls++
			dailyStats.NationalTime += billSec
			dailyStats.NationalCost += roundedPrice
		} else {
			totals.InternationalCalls++
			totals.InternationalTime += billSec
			totals.InternationalCost += roundedPrice

			dailyStats.InternationalCalls++
			dailyStats.InternationalTime += billSec
			dailyStats.InternationalCost += roundedPrice
		}

		// Update Portuguese call counts if it's a Portuguese number
		if ptCallType != nil && isNational {
			switch ptCallType.Type {
			case "landline":
				totals.LandlineCalls++
				totals.LandlineCost += roundedPrice
				dailyStats.LandlineCalls++
				dailyStats.LandlineCost += roundedPrice
			case "mobile":
				totals.MobileCalls++
				totals.MobileCost += roundedPrice
				dailyStats.MobileCalls++
				dailyStats.MobileCost += roundedPrice
			case "premium":
				totals.PremiumCalls++
				totals.PremiumCost += roundedPrice
				dailyStats.PremiumCalls++
				dailyStats.PremiumCost += roundedPrice
			case "free":
				totals.FreeCalls++
				totals.FreeCost += roundedPrice
				dailyStats.FreeCalls++
				dailyStats.FreeCost += roundedPrice
			case "shared_cost":
				totals.SharedCostCalls++
				totals.SharedCostCost += roundedPrice
				dailyStats.SharedCostCalls++
				dailyStats.SharedCostCost += roundedPrice
			case "internet":
				totals.InternetCalls++
				totals.InternetCost += roundedPrice
				dailyStats.InternetCalls++
				dailyStats.InternetCost += roundedPrice
			case "audiotext":
				totals.AudiotextCalls++
				totals.AudiotextCost += roundedPrice
				dailyStats.AudiotextCalls++
				dailyStats.AudiotextCost += roundedPrice
			case "special_service":
				totals.SpecialServiceCalls++
				totals.SpecialServiceCost += roundedPrice
				dailyStats.SpecialServiceCalls++
				dailyStats.SpecialServiceCost += roundedPrice
			}
		}

		// Update caller statistics based on caller type
		billMinutes := billSec / 60

		switch callerType.Type {
		case "geographic":
			// Update geographic caller stats
			var geoStats *GeographicCallerStats
			found := false
			for _, gs := range totals.GeographicCallers {
				if gs.CallerNumber == src {
					geoStats = gs
					found = true
					break
				}
			}
			if !found {
				geoStats = &GeographicCallerStats{
					CallerNumber: src,
				}
				totals.GeographicCallers = append(totals.GeographicCallers, geoStats)
			}

			geoStats.TotalCalls++
			geoStats.TotalMinutes += billMinutes

			// Classify destination and update appropriate counters
			switch destinationType.Type {
			case "landline":
				geoStats.LandlineCalls++
				geoStats.LandlineMinutes += billMinutes
			case "mobile":
				geoStats.MobileCalls++
				geoStats.MobileMinutes += billMinutes
			case "international":
				geoStats.InternationalCalls++
				geoStats.InternationalMinutes += billMinutes
			case "non_geographic":
				geoStats.NonGeographicCalls++
				geoStats.NonGeographicMinutes += billMinutes
			case "short":
				geoStats.ShortCalls++
				geoStats.ShortMinutes += billMinutes
			case "value_added":
				geoStats.ValueAddedCalls++
				geoStats.ValueAddedMinutes += billMinutes

				// Check if it's specifically a 760 number
				if IsValue760Number(dst) {
					geoStats.Value760Calls++
					geoStats.Value760Minutes += billMinutes
				}
			default:
				// Nomad or other types
				if strings.Contains(destinationType.Description, "nomad") {
					geoStats.NomadCalls++
					geoStats.NomadMinutes += billMinutes
				}
			}

		case "nomad":
			// Update nomad caller stats
			var nomadStats *NomadCallerStats
			found := false
			for _, ns := range totals.NomadCallers {
				if ns.CallerNumber == src {
					nomadStats = ns
					found = true
					break
				}
			}
			if !found {
				nomadStats = &NomadCallerStats{
					CallerNumber: src,
				}
				totals.NomadCallers = append(totals.NomadCallers, nomadStats)
			}

			nomadStats.TotalCalls++
			nomadStats.TotalMinutes += billMinutes

			// Track international calls for nomad callers
			if destinationType.Type == "international" {
				nomadStats.InternationalCalls++
				nomadStats.InternationalMinutes += billMinutes
			}
		}

		// Update remaining time tracking for daily stats
		dailyStats.RemainingTimeAtStart = remainingAtStart
		dailyStats.RemainingTimeAtEnd = totals.RemainingTime

		// Add call details with Portuguese call type information
		callDetails = append(callDetails, CallDetail{
			Sequence:           call.Value.Sequence,
			CdrId:              call.Value.CdrId,
			UniqueId:           call.Value.UniqueId,
			CallPrice:          roundedPrice,
			CountryName:        countryName,
			CountryCode:        region,
			CallType:           callType,
			InPlan:             inPlan,
			IsNational:         isNational,
			PortugueseCallType: ptCallType,
			CallerType:         &callerType,
			DestinationType:    &destinationType,
		})

		// Update total costs and time
		totals.TotalCost += roundedPrice
		totals.TotalTime += billSec

		dailyStats.TotalCost += roundedPrice
		dailyStats.TotalTime += billSec
	}

	// Convert client map to slice and add daily statistics
	clients := make([]ClientSummary, 0, len(clientTotals))
	for clientID, totals := range clientTotals {
		if dailyStatsMap, exists := clientDailyStats[clientID]; exists {
			dailyStats := make([]DailySummary, 0, len(dailyStatsMap))
			for _, daily := range dailyStatsMap {
				dailyStats = append(dailyStats, *daily)
			}

			// Sort daily stats by date
			sort.Slice(dailyStats, func(i, j int) bool {
				return dailyStats[i].Date < dailyStats[j].Date
			})

			totals.DailyStats = dailyStats
		}

		clients = append(clients, *totals)
	}

	return CalculatePriceFullResponse{
		Clients: clients,
		Calls:   callDetails,
	}
}

func GetPortugueseCallType(number string) PortugueseCallType {
	parsed, err := phonenumbers.Parse(number, "PT")
	if err != nil {
		return PortugueseCallType{
			Type:        "unknown",
			Description: "Invalid phone number",
			Prefix:      "",
			Category:    "unknown",
		}
	}

	region := phonenumbers.GetRegionCodeForNumber(parsed)
	if region != "PT" {
		return PortugueseCallType{
			Type:        "foreign",
			Description: "Non-Portuguese number",
			Prefix:      "",
			Category:    "foreign",
		}
	}

	nationalNumber := strconv.FormatUint(parsed.GetNationalNumber(), 10)

	if len(nationalNumber) != 9 {
		return PortugueseCallType{
			Type:        "unknown",
			Description: "Invalid Portuguese number format",
			Prefix:      "",
			Category:    "unknown",
		}
	}

	firstDigit := string(nationalNumber[0])

	switch firstDigit {
	case "2":
		return PortugueseCallType{
			Type:        "landline",
			Description: "Fixed line (landline)",
			Prefix:      "2",
			Category:    "standard",
		}
	case "3":
		return PortugueseCallType{
			Type:        "special_service",
			Description: "Special service number",
			Prefix:      "3",
			Category:    "special_service",
		}
	case "6":
		if len(nationalNumber) >= 3 {
			prefix3 := nationalNumber[:3]
			switch prefix3 {
			case "646":
				return PortugueseCallType{
					Type:        "audiotext",
					Description: "Audiotext/Entertainment service",
					Prefix:      "646",
					Category:    "premium_rate",
				}
			case "671":
				return PortugueseCallType{
					Type:        "internet",
					Description: "Internet access service",
					Prefix:      "671",
					Category:    "special_service",
				}
			default:
				return PortugueseCallType{
					Type:        "premium",
					Description: "Premium rate service",
					Prefix:      "6",
					Category:    "premium_rate",
				}
			}
		}
		return PortugueseCallType{
			Type:        "premium",
			Description: "Premium rate service",
			Prefix:      "6",
			Category:    "premium_rate",
		}
	case "7":
		if len(nationalNumber) >= 3 {
			prefix3 := nationalNumber[:3]
			switch prefix3 {
			case "707", "708":
				return PortugueseCallType{
					Type:        "shared_cost",
					Description: "Shared cost service",
					Prefix:      prefix3,
					Category:    "special_service",
				}
			case "760":
				return PortugueseCallType{
					Type:        "shared_cost",
					Description: "Shared cost service",
					Prefix:      "760",
					Category:    "special_service",
				}
			}
		}
		return PortugueseCallType{
			Type:        "special_service",
			Description: "Special service number",
			Prefix:      "7",
			Category:    "special_service",
		}
	case "8":
		if len(nationalNumber) >= 3 {
			prefix3 := nationalNumber[:3]
			switch prefix3 {
			case "800":
				return PortugueseCallType{
					Type:        "free",
					Description: "Toll-free number",
					Prefix:      "800",
					Category:    "free_service",
				}
			case "808":
				return PortugueseCallType{
					Type:        "shared_cost",
					Description: "Shared cost service",
					Prefix:      "808",
					Category:    "special_service",
				}
			case "809":
				return PortugueseCallType{
					Type:        "special_service",
					Description: "Special service number",
					Prefix:      "809",
					Category:    "special_service",
				}
			}
		}
		return PortugueseCallType{
			Type:        "special_service",
			Description: "Special service number",
			Prefix:      "8",
			Category:    "special_service",
		}
	case "9":
		if len(nationalNumber) >= 2 {
			prefix2 := nationalNumber[:2]
			switch prefix2 {
			case "91":
				return PortugueseCallType{
					Type:        "mobile",
					Description: "Mobile (Vodafone/original Telecel)",
					Prefix:      "91",
					Category:    "mobile",
				}
			case "92":
				return PortugueseCallType{
					Type:        "mobile",
					Description: "Mobile (MEO/TMN)",
					Prefix:      "92",
					Category:    "mobile",
				}
			case "93":
				return PortugueseCallType{
					Type:        "mobile",
					Description: "Mobile (NOS/Optimus)",
					Prefix:      "93",
					Category:    "mobile",
				}
			case "96":
				return PortugueseCallType{
					Type:        "mobile",
					Description: "Mobile (MEO/TMN)",
					Prefix:      "96",
					Category:    "mobile",
				}
			default:
				return PortugueseCallType{
					Type:        "mobile",
					Description: "Mobile number",
					Prefix:      "9",
					Category:    "mobile",
				}
			}
		}
		return PortugueseCallType{
			Type:        "mobile",
			Description: "Mobile number",
			Prefix:      "9",
			Category:    "mobile",
		}
	default:
		return PortugueseCallType{
			Type:        "unknown",
			Description: "Unknown Portuguese number type",
			Prefix:      firstDigit,
			Category:    "unknown",
		}
	}
}

// ClassifyNumber classifies a phone number based on the provided rules
func ClassifyNumber(number string) CallClassification {
	// Clean the number
	cleanNumber := strings.TrimSpace(number)

	// Remove + prefix for length calculation
	numberForLength := strings.TrimPrefix(cleanNumber, "+")
	numberForLength = strings.TrimPrefix(numberForLength, "00")

	length := len(numberForLength)

	// Short codes (less than 9 digits)
	if length < 9 {
		return CallClassification{
			Type:        "short",
			Description: "Short code",
		}
	}

	// International calls (more than 9 digits)
	if length > 9 {
		return CallClassification{
			Type:        "international",
			Description: "International call",
		}
	}

	// At this point, length == 9, so it's a Portuguese number
	// Get the first digit(s) to classify
	var firstDigit string
	var firstThreeDigits string

	if strings.HasPrefix(cleanNumber, "+3512") && len(cleanNumber) == 13 {
		// +3512XXXXXXX format (landline)
		firstDigit = "2"
		if len(cleanNumber) >= 8 {
			firstThreeDigits = cleanNumber[5:8]
		}
	} else if strings.HasPrefix(cleanNumber, "003512") && len(cleanNumber) == 14 {
		// 003512XXXXXXX format (landline)
		firstDigit = "2"
		if len(cleanNumber) >= 9 {
			firstThreeDigits = cleanNumber[6:9]
		}
	} else if strings.HasPrefix(cleanNumber, "3512") && len(cleanNumber) == 12 {
		// 3512XXXXXXX format (landline)
		firstDigit = "2"
		if len(cleanNumber) >= 7 {
			firstThreeDigits = cleanNumber[4:7]
		}
	} else if strings.HasPrefix(cleanNumber, "+3519") && len(cleanNumber) == 13 {
		// +3519XXXXXXX format (mobile)
		firstDigit = "9"
		if len(cleanNumber) >= 8 {
			firstThreeDigits = cleanNumber[5:8]
		}
	} else if strings.HasPrefix(cleanNumber, "003519") && len(cleanNumber) == 14 {
		// 003519XXXXXXX format (mobile)
		firstDigit = "9"
		if len(cleanNumber) >= 9 {
			firstThreeDigits = cleanNumber[6:9]
		}
	} else if strings.HasPrefix(cleanNumber, "3519") && len(cleanNumber) == 12 {
		// 3519XXXXXXX format (mobile)
		firstDigit = "9"
		if len(cleanNumber) >= 7 {
			firstThreeDigits = cleanNumber[4:7]
		}
	} else if len(numberForLength) == 9 {
		// National format (9 digits)
		firstDigit = string(numberForLength[0])
		if len(numberForLength) >= 3 {
			firstThreeDigits = numberForLength[0:3]
		}
	} else {
		// Fallback
		firstDigit = string(numberForLength[0])
		if len(numberForLength) >= 3 {
			firstThreeDigits = numberForLength[0:3]
		}
	}

	// Value-added numbers (760, 761)
	if firstThreeDigits == "760" || firstThreeDigits == "761" {
		return CallClassification{
			Type:        "value_added",
			Description: "Value-added service (760/761)",
		}
	}

	// Landline (starts with 2)
	if firstDigit == "2" {
		return CallClassification{
			Type:        "landline",
			Description: "Portuguese landline",
		}
	}

	// Mobile (starts with 9)
	if firstDigit == "9" {
		return CallClassification{
			Type:        "mobile",
			Description: "Portuguese mobile",
		}
	}

	// Non-geographic numbers (starts with 3, 7, or 8)
	// Exception: 760 and 761 are already handled above
	if firstDigit == "3" {
		return CallClassification{
			Type:        "non_geographic",
			Description: "Non-geographic number (3xx)",
		}
	}

	if firstDigit == "8" {
		return CallClassification{
			Type:        "non_geographic",
			Description: "Non-geographic number (8xx)",
		}
	}

	if firstDigit == "7" {
		// 7xx but not 760/761
		return CallClassification{
			Type:        "non_geographic",
			Description: "Non-geographic number (7xx)",
		}
	}

	// Default to other
	return CallClassification{
		Type:        "other",
		Description: "Other number type",
	}
}

// ClassifyCallerType determines if a caller is geographic (landline), nomad, or other
func ClassifyCallerType(number string) CallerType {
	// Clean the number
	cleanNumber := strings.TrimSpace(number)

	// Classify the number
	classification := ClassifyNumber(cleanNumber)

	callerType := CallerType{
		Number:         cleanNumber,
		Classification: classification.Type,
	}

	// Geographic = landline
	switch classification.Type {
	case "landline":
		callerType.Type = "geographic"
	case "non_geographic":
		// Nomad numbers are a subset of non-geographic
		// For simplicity, we'll consider all non-geographic as potentially nomad
		// You can refine this logic based on specific prefixes if needed
		callerType.Type = "nomad"
	default:
		callerType.Type = "other"
	}

	return callerType
}

// IsValue760Number checks if a number specifically starts with 760
func IsValue760Number(number string) bool {
	cleanNumber := strings.TrimSpace(number)

	// Remove + prefix and country code
	numberForCheck := strings.TrimPrefix(cleanNumber, "+")
	numberForCheck = strings.TrimPrefix(numberForCheck, "00")

	// Check different formats
	if strings.HasPrefix(cleanNumber, "+351760") && len(cleanNumber) == 13 {
		return true
	}
	if strings.HasPrefix(cleanNumber, "00351760") && len(cleanNumber) == 14 {
		return true
	}
	if strings.HasPrefix(cleanNumber, "351760") && len(cleanNumber) == 12 {
		return true
	}
	if strings.HasPrefix(numberForCheck, "760") && len(numberForCheck) == 9 {
		return true
	}

	return false
}
