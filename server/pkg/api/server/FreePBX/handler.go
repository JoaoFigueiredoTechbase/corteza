package freepbx

import (
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
