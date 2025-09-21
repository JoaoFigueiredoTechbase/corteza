package freepbx

import (
	"encoding/json"
	"fmt"
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

func BuildPriceMap(prices []KV[PriceValue]) map[string]PriceValue {
	priceMap := make(map[string]PriceValue)
	for _, kv := range prices {
		key := kv.Value.CountryCode + "_" + kv.Value.Type
		priceMap[key] = kv.Value
	}
	return priceMap
}

func GetNumberInfo(number string) (region string, isMobile bool, err error) {
	parsed, err := phonenumbers.Parse(number, "")
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

func BuildCalculatePriceResponses(calls []KV[CallValue], priceMap map[string]PriceValue) []CalculatePriceResponse {
	responses := make([]CalculatePriceResponse, 0, len(calls))

	for _, call := range calls {
		dst := call.Value.Dst
		billSec, err := strconv.Atoi(call.Value.BillSec)
		if err != nil {
			continue
		}

		region, isMobile, err := GetNumberInfo(dst)
		if err != nil {
			continue
		}

		priceEntry, ok := GetCallPrice(priceMap, region, isMobile)
		if !ok {
			continue
		}

		pricePerMinute, err := strconv.ParseFloat(priceEntry.Price, 64)
		if err != nil {
			continue
		}

		price, err := CalculateCallPrice(billSec, priceEntry.CallRating, pricePerMinute)
		if err != nil {
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
			PriceRecord: priceEntry.PriceRecord,
		})
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
