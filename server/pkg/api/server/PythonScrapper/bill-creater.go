package PythonScrapper

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"unicode/utf8"
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

type BillRequest struct {
	Email  string          `json:"email"`
	Senha  string          `json:"senha"`
	Avenca json.RawMessage `json:"avenca"` // still raw JSON string
}

func ParseOrders(data []byte) ([]Order, error) {
	log.Printf("ParseOrders: Raw input data: %s", string(data))

	var kvs []KV
	if err := json.Unmarshal(data, &kvs); err != nil {
		return nil, err
	}

	log.Printf("ParseOrders: Parsed %d KV entries", len(kvs))

	var orders []Order
	for i, kv := range kvs {
		log.Printf("ParseOrders: Processing KV %d - IdClient: %s, Address: %s", i, kv.Value.IdClient, kv.Value.Address)
		log.Printf("ParseOrders: Raw Products string: %s", kv.Value.Products)

		order := Order{
			Address:  kv.Value.Address,
			DocDate:  kv.Value.DocDate,
			IdClient: kv.Value.IdClient,
		}

		// Fix products string: wrap multiple arrays into one
		productsStr := kv.Value.Products
		log.Printf("ParseOrders: Original products string: %s", productsStr)

		// Example: "[{...}],[{...},{...}]" -> "[{...},{...},{...}]"
		if strings.Contains(productsStr, "],[") {
			productsStr = strings.ReplaceAll(productsStr, "],[", ",")
			log.Printf("ParseOrders: Fixed products string: %s", productsStr)
		}

		var rawProducts []struct {
			Value ProductBill `json:"@value"`
		}
		if err := json.Unmarshal([]byte(productsStr), &rawProducts); err != nil {
			log.Printf("ParseOrders: Error parsing products: %v", err)
			return nil, fmt.Errorf("error parsing products for client %s: %w", kv.Value.IdClient, err)
		}

		log.Printf("ParseOrders: Parsed %d raw products", len(rawProducts))

		for j, p := range rawProducts {
			log.Printf("ParseOrders: Product %d: %+v", j, p.Value)
			order.Products = append(order.Products, p.Value)
		}

		log.Printf("ParseOrders: Final order has %d products: %+v", len(order.Products), order)
		orders = append(orders, order)
	}

	// Remove duplicate details from products across all orders
	orders = removeDuplicateProductDetails(orders)

	log.Printf("ParseOrders: Returning %d orders", len(orders))
	return orders, nil
}

// removeDuplicateProductDetails removes details from products that appear earlier
// if the same details appear in a later product
func removeDuplicateProductDetails(orders []Order) []Order {
	// First, collect all products with their positions
	type ProductPosition struct {
		OrderIndex   int
		ProductIndex int
		Product      *ProductBill
	}

	var allProducts []ProductPosition

	// Collect all products with their positions
	for orderIdx, order := range orders {
		for productIdx := range order.Products {
			allProducts = append(allProducts, ProductPosition{
				OrderIndex:   orderIdx,
				ProductIndex: productIdx,
				Product:      &orders[orderIdx].Products[productIdx],
			})
		}
	}

	// Find which details appear multiple times and mark earlier ones for removal
	detailsLastSeen := make(map[string]int) // maps detail -> last index where it appears

	// First pass: find the last occurrence of each detail
	for i, pp := range allProducts {
		details := pp.Product.Details
		if details != "" {
			detailsLastSeen[details] = i
		}
	}

	// Second pass: clear details from earlier occurrences
	for i, pp := range allProducts {
		details := pp.Product.Details
		if details != "" {
			if lastIndex, exists := detailsLastSeen[details]; exists && i < lastIndex {
				// This is not the last occurrence, clear the details
				pp.Product.Details = ""
				log.Printf("Cleared details '%s' from product at order %d, product %d",
					details, pp.OrderIndex, pp.ProductIndex)
			}
		}
	}

	return orders
}

func HandleBillCreation(w http.ResponseWriter, r *http.Request) {
	log.Println("Received Bill creation request")

	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("X-Content-Type-Options", "nosniff")

	defer r.Body.Close()

	body, err := io.ReadAll(r.Body)
	if err != nil {
		resp := Response{Success: false, Error: fmt.Sprintf("Failed to read request body: %v", err)}
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(resp)
		return
	}

	// Parse wrapper request
	var req BillRequest
	if err := json.Unmarshal(body, &req); err != nil {
		resp := Response{Success: false, Error: fmt.Sprintf("Invalid request: %v", err)}
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(resp)
		return
	}

	// Now parse avenca -> orders
	orders, err := ParseOrders(req.Avenca)
	if err != nil {
		resp := Response{Success: false, Error: fmt.Sprintf("Failed to parse orders: %v", err)}
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(resp)
		return
	}

	// Debug
	log.Printf("Parsed %d orders for email=%s", len(orders), req.Email)

	// Prepare script execution
	cwd, _ := os.Getwd()
	scriptPath := filepath.Join(cwd, "pkg", "api", "server", "PythonScrapper", "python", "bill-creator.py")

	if _, err := os.Stat(scriptPath); os.IsNotExist(err) {
		resp := Response{Success: false, Error: "Python script not found"}
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(resp)
		return
	}

	// Convert orders to JSON
	ordersJSON, err := json.Marshal(orders)
	if err != nil {
		resp := Response{Success: false, Error: fmt.Sprintf("Failed to serialize orders: %v", err)}
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(resp)
		return
	}

	// Create context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), MaxScriptTimeout)
	defer cancel()

	// Send email, senha, and orders as arguments
	cmd := exec.CommandContext(ctx, "py", scriptPath, req.Email, req.Senha, string(ordersJSON))

	cmd.Env = append(os.Environ(),
		"PYTHONIOENCODING=utf-8",
		"PYTHONUNBUFFERED=1",
	)

	var stdout, stderr strings.Builder
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err = cmd.Run()

	if stderr.Len() > 0 {
		log.Printf("Python script stderr: %s", stderr.String())
	}
	output := []byte(stdout.String())

	var resp Response

	// Check for timeout
	if ctx.Err() == context.DeadlineExceeded {
		resp = Response{
			Success: false,
			Error:   "Script execution timed out",
		}
		w.WriteHeader(http.StatusRequestTimeout)
		json.NewEncoder(w).Encode(resp)
		return
	}

	// Check for execution errors
	if err != nil {
		// Include stderr in error message for better debugging
		errorMsg := fmt.Sprintf("Failed to execute Python script: %v", err)
		if stderr.Len() > 0 {
			errorMsg += fmt.Sprintf(" | stderr: %s", stderr.String())
		}

		resp = Response{
			Success: false,
			Error:   errorMsg,
		}
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(resp)
		return
	}

	// Check if we got any output
	if len(output) == 0 {
		resp = Response{
			Success: false,
			Error:   fmt.Sprintf("No output from Python script | stderr: %s", stderr.String()),
		}
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(resp)
		return
	}

	// Ensure output is valid UTF-8
	outputStr := string(output)
	if !utf8.ValidString(outputStr) {
		outputStr = strings.ToValidUTF8(outputStr, "")
		output = []byte(outputStr)
	}

	// Parse Python script output - use a more flexible structure
	var pyData map[string]interface{}
	if err := json.Unmarshal(output, &pyData); err != nil {
		resp := Response{
			Success: false,
			Error:   fmt.Sprintf("Failed to decode Python output: %v | Raw output: %s | stderr: %s", err, string(output), stderr.String()),
		}
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(resp)
		return
	}

	// Handle Python script errors
	success, ok := pyData["success"].(bool)
	if !ok || !success {
		errorMsg := "Python script reported failure"
		if errStr, exists := pyData["error"].(string); exists {
			errorMsg = errStr
		}
		resp := Response{
			Success: false,
			Error:   errorMsg,
		}
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(resp)
		return
	}

	// Return the complete Python script response
	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(pyData); err != nil {
		log.Printf("Failed to encode response: %v", err)
	}
}
