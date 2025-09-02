package PythonScrapper

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"
)

type KV struct {
	Value KVValue `json:"@value"`
	Type  string  `json:"@type"`
}

type KVValue struct {
	Address  string `json:"Address"`
	DocDate  string `json:"DocDate"`
	IdClient string `json:"IdClient"`
	Products string `json:"Products"` // raw string containing products JSON
}

type ProductBill struct {
	Details   string `json:"Details,omitempty"`
	Discount  string `json:"Discount"`
	IdProduct string `json:"IdProduct"`
	Price     string `json:"Price"`
	Quantity  string `json:"Quantity"`
	Tax       string `json:"Tax"`
}

type Order struct {
	Address  string        `json:"Address"`
	DocDate  string        `json:"DocDate"`
	IdClient string        `json:"IdClient"`
	Products []ProductBill `json:"Products"`
}

func ParseOrders(data []byte) ([]Order, error) {
	var kvs []KV
	if err := json.Unmarshal(data, &kvs); err != nil {
		return nil, err
	}

	var orders []Order
	for _, kv := range kvs {
		order := Order{
			Address:  kv.Value.Address,
			DocDate:  kv.Value.DocDate,
			IdClient: kv.Value.IdClient,
		}

		// Fix products string: wrap multiple arrays into one
		productsStr := kv.Value.Products
		// Example: "[{...}],[{...},{...}]" -> "[{...},{...},{...}]"
		if strings.Contains(productsStr, "],[") {
			productsStr = strings.ReplaceAll(productsStr, "],[", ",")
		}

		var rawProducts []struct {
			Value ProductBill `json:"@value"`
		}
		if err := json.Unmarshal([]byte(productsStr), &rawProducts); err != nil {
			return nil, fmt.Errorf("error parsing products for client %s: %w", kv.Value.IdClient, err)
		}

		for _, p := range rawProducts {
			order.Products = append(order.Products, p.Value)
		}

		orders = append(orders, order)
	}
	return orders, nil
}

func HandleBillCreation(w http.ResponseWriter, r *http.Request) {
	log.Println("Received Bill creation request")

	defer r.Body.Close()

	body, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "Failed to read request body", http.StatusBadRequest)
		log.Println("Error reading body:", err)
		return
	}

	orders, err := ParseOrders(body)
	if err != nil {
		http.Error(w, "Failed to parse orders: "+err.Error(), http.StatusBadRequest)
		log.Println("Error parsing orders:", err)
		return
	}

	// Log the orders
	for _, o := range orders {
		log.Printf("Client %s (%s): %d products\n", o.IdClient, o.Address, len(o.Products))
		for _, p := range o.Products {
			log.Printf("  -> %+v\n", p)
		}
	}

	// Return the orders as JSON
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(orders); err != nil {
		log.Println("Error encoding response:", err)
		http.Error(w, "Failed to encode response", http.StatusInternalServerError)
	}
}
