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

	log.Printf("ParseOrders: Returning %d orders", len(orders))
	return orders, nil
}

func HandleBillCreation(w http.ResponseWriter, r *http.Request) {
	log.Println("Received Bill creation request")

	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("X-Content-Type-Options", "nosniff")

	defer r.Body.Close()

	body, err := io.ReadAll(r.Body)
	if err != nil {
		resp := Response{
			Success: false,
			Error:   fmt.Sprintf("Failed to read request body: %v", err),
		}
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(resp)
		return
	}

	orders, err := ParseOrders(body)
	if err != nil {
		resp := Response{
			Success: false,
			Error:   fmt.Sprintf("Failed to parse orders: %v", err),
		}
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(resp)
		return
	}

	// Log the orders (keep your existing logic)
	for _, o := range orders {
		log.Printf("Client %s (%s): %d products\n", o.IdClient, o.Address, len(o.Products))
		for _, p := range o.Products {
			log.Printf("  -> %+v\n", p)
		}
	}

	// Prepare script execution (similar to HandleScrapeKeyInvoiceProducts)
	cwd, _ := os.Getwd()
	scriptPath := filepath.Join(cwd, "pkg", "api", "server", "PythonScrapper", "python", "bill-creator.py") // Adjust script name as needed

	// Check if script exists
	if _, err := os.Stat(scriptPath); os.IsNotExist(err) {
		resp := Response{
			Success: false,
			Error:   "Python script not found",
		}
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(resp)
		return
	}

	// Convert orders to JSON string to pass as argument to Python script
	ordersJSON, err := json.Marshal(orders)
	if err != nil {
		resp := Response{
			Success: false,
			Error:   fmt.Sprintf("Failed to serialize orders: %v", err),
		}
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(resp)
		return
	}

	// Create context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), MaxScriptTimeout)
	defer cancel()

	// Execute Python script with timeout, passing orders as JSON argument
	cmd := exec.CommandContext(ctx, "py", scriptPath, string(ordersJSON))

	// Set environment variables for better Python execution
	cmd.Env = append(os.Environ(),
		"PYTHONIOENCODING=utf-8",
		"PYTHONUNBUFFERED=1",
	)

	// Capture both stdout and stderr separately for debugging
	var stdout, stderr strings.Builder
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err = cmd.Run()

	// Log stderr for debugging (contains our debug messages)
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
	return

	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(resp); err != nil {
		log.Printf("Failed to encode response: %v", err)
	}
}
