package freepbx

import (
	"encoding/json"
	"fmt"
	"log"
	"math"
	"strconv"
	"strings"

	"github.com/nyaruka/phonenumbers"
)

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
		var clientArray []KV[ClientValue]
		if err := json.Unmarshal([]byte(clientStr), &clientArray); err != nil {
			return nil, fmt.Errorf("failed to parse clients JSON: %v", err)
		}
		clients = append(clients, clientArray...)
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

func BuildCalculatePriceResponses(calls []KV[CallValue], priceMap map[string]PriceValue, clientMap map[string]ClientValue) []CalculatePriceResponse {
	responses := make([]CalculatePriceResponse, 0, len(calls))

	for _, call := range calls {
		dst := call.Value.Dst
		trunkClientRecord := call.Value.TrunkClientRecord
		billSec, err := strconv.Atoi(call.Value.BillSec)
		if err != nil {
			log.Printf("DEBUG: Invalid billsec for call %s: %v", call.Value.Sequence, err)
			continue
		}

		// Check if client exists
		client, clientExists := clientMap[trunkClientRecord]
		if !clientExists {
			log.Printf("DEBUG: Client not found for trunk_cliente_record %s", trunkClientRecord)
			continue
		}

		region, isMobile, err := GetNumberInfo(dst)
		if err != nil {
			log.Printf("DEBUG: Failed to parse number %s for call %s: %v", dst, call.Value.Sequence, err)
			continue
		}

		priceEntry, ok := GetCallPrice(priceMap, region, isMobile)
		if !ok {
			log.Printf("DEBUG: No price found for region %s, mobile %t (call %s)", region, isMobile, call.Value.Sequence)
			continue
		}

		pricePerMinute, err := strconv.ParseFloat(priceEntry.Price, 64)
		if err != nil {
			log.Printf("DEBUG: Invalid price %s for call %s: %v", priceEntry.Price, call.Value.Sequence, err)
			continue
		}

		price, err := CalculateCallPrice(billSec, priceEntry.CallRating, pricePerMinute)
		if err != nil {
			log.Printf("DEBUG: Failed to calculate price for call %s: %v", call.Value.Sequence, err)
			continue
		}

		callType := "other"
		if isMobile {
			callType = "mobile"
		}

		responses = append(responses, CalculatePriceResponse{
			Sequence:    call.Value.Sequence,
			CallType:    callType,
			CallPrice:   price,
			PriceRecord: fmt.Sprintf("%s_%s", priceEntry.CountryCode, priceEntry.Type),
		})

		log.Printf("DEBUG: Successfully calculated price for call %s: client=%s, region=%s, mobile=%t, price=%.4f",
			call.Value.Sequence, client.ClientRecord, region, isMobile, price)
	}

	return responses
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

func BuildFullPriceResponses(calls []KV[CallValue], priceMap map[string]PriceValue, clientMap map[string]ClientValue) CalculatePriceFullResponse {
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
		callPrice, err := CalculateCallPrice(billSec, priceEntry.CallRating, pricePerMinute)
		if err != nil {
			log.Printf("DEBUG: Price calc failed for %s: %v", call.Value.Sequence, err)
			continue
		}

		callType := "other"
		if isMobile {
			callType = "mobile"
		}

		// Add to call details
		callDetails = append(callDetails, CallDetail{
			Sequence:    call.Value.Sequence,
			CdrId:       call.Value.CdrId,
			UniqueId:    call.Value.UniqueId,
			CallPrice:   math.Round(callPrice*100) / 100,
			CountryName: countryName,
			CountryCode: region,
			CallType:    callType,
		})

		// Update client totals
		totals, exists := clientTotals[client.RecordID]
		if !exists {
			totals = &ClientSummary{
				ClientRecord: client.ClientRecord,
				RecordID:     client.RecordID,
			}
			clientTotals[client.RecordID] = totals
		}

		totals.TotalCost += math.Round(callPrice*100) / 100
		totals.TotalTime += billSec

		if region == "PT" { // 🇵🇹 replace with your "national" region code
			totals.NationalCost += math.Round(callPrice*100) / 100
			totals.NationalTime += billSec
		} else {
			totals.InternationalCost += math.Round(callPrice*100) / 100
			totals.InternationalTime += billSec
		}
	}

	// Convert client map → slice
	clients := make([]ClientSummary, 0, len(clientTotals))
	for _, c := range clientTotals {
		clients = append(clients, *c)
	}

	return CalculatePriceFullResponse{
		Clients: clients,
		Calls:   callDetails,
	}
}
