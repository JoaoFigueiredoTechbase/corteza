package PythonScrapper

import (
	"context"
	"encoding/base64"
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

type PythonBillData struct {
	BillID        string  `json:"bill_id"`
	ClientID      string  `json:"client_id"`
	TotalAmount   float64 `json:"total_amount"`
	Status        string  `json:"status"`
	Error         string  `json:"error,omitempty"`
	ProductsCount int     `json:"products_count"`
	CreationDate  string  `json:"creation_date"`
	PDFFilename   string  `json:"pdf_filename"`
	PDFContent    string  `json:"pdf_content"` // Base64 encoded
}

type PythonPDFFile struct {
	ClientID string `json:"client_id"`
	BillID   string `json:"bill_id"`
	Filename string `json:"filename"`
	Content  string `json:"content"` // Base64 encoded
}

type PythonSummary struct {
	TotalOrdersProcessed int     `json:"total_orders_processed"`
	SuccessfulBills      int     `json:"successful_bills"`
	FailedBills          int     `json:"failed_bills"`
	TotalRevenue         float64 `json:"total_revenue"`
	ProcessingTime       string  `json:"processing_time"`
}

type PythonScriptResponse struct {
	Success bool              `json:"success"`
	Data    *PythonScriptData `json:"data,omitempty"`
	Error   string            `json:"error,omitempty"`
	Message string            `json:"message,omitempty"`
}

type PythonScriptData struct {
	Summary  PythonSummary    `json:"summary"`
	Bills    []PythonBillData `json:"bills"`
	PDFFiles []PythonPDFFile  `json:"pdf_files"`
}

// Enhanced response structure for our API
type BillCreationResponse struct {
	Success bool               `json:"success"`
	Message string             `json:"message,omitempty"`
	Error   string             `json:"error,omitempty"`
	Summary *PythonSummary     `json:"summary,omitempty"`
	Bills   []EnhancedBillData `json:"bills,omitempty"`
}

type EnhancedBillData struct {
	BillID        string       `json:"bill_id"`
	ClientID      string       `json:"client_id"`
	TotalAmount   float64      `json:"total_amount"`
	Status        string       `json:"status"`
	Error         string       `json:"error,omitempty"`
	ProductsCount int          `json:"products_count"`
	CreationDate  string       `json:"creation_date"`
	PDFFile       *PDFFileInfo `json:"pdf_file,omitempty"`
}

type PDFFileInfo struct {
	Filename    string `json:"filename"`
	Content     string `json:"content"`      // Base64 encoded PDF content
	Size        int    `json:"size"`         // Size in bytes
	ContentType string `json:"content_type"` // Always "application/pdf"
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
		sendErrorResponse(w, http.StatusBadRequest, fmt.Sprintf("Failed to read request body: %v", err))
		return
	}

	// Parse wrapper request
	var req BillRequest
	if err := json.Unmarshal(body, &req); err != nil {
		sendErrorResponse(w, http.StatusBadRequest, fmt.Sprintf("Invalid request: %v", err))
		return
	}

	// Now parse avenca -> orders
	orders, err := ParseOrders(req.Avenca)
	if err != nil {
		sendErrorResponse(w, http.StatusBadRequest, fmt.Sprintf("Failed to parse orders: %v", err))
		return
	}

	// Debug
	log.Printf("Parsed %d orders for email=%s", len(orders), req.Email)

	// Prepare script execution
	cwd, _ := os.Getwd()
	scriptPath := filepath.Join(cwd, "pkg", "api", "server", "PythonScrapper", "python", "bill-creator.py")

	if _, err := os.Stat(scriptPath); os.IsNotExist(err) {
		sendErrorResponse(w, http.StatusInternalServerError, "Python script not found")
		return
	}

	// Convert orders to JSON
	ordersJSON, err := json.Marshal(orders)
	if err != nil {
		sendErrorResponse(w, http.StatusInternalServerError, fmt.Sprintf("Failed to serialize orders: %v", err))
		return
	}

	// Create context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), MaxScriptTimeout)
	defer cancel()

	// Send email, senha, and orders as arguments
	pythonExec := "python3"
	if _, err := exec.LookPath("py"); err == nil {
		pythonExec = "py"
	}

	// Execute Python script with timeout
	cmd := exec.CommandContext(ctx, pythonExec, scriptPath, req.Email, req.Senha, string(ordersJSON))

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

	// Check for timeout
	if ctx.Err() == context.DeadlineExceeded {
		sendErrorResponse(w, http.StatusRequestTimeout, "Script execution timed out")
		return
	}

	// Check for execution errors
	if err != nil {
		errorMsg := fmt.Sprintf("Failed to execute Python script: %v", err)
		if stderr.Len() > 0 {
			errorMsg += fmt.Sprintf(" | stderr: %s", stderr.String())
		}
		sendErrorResponse(w, http.StatusInternalServerError, errorMsg)
		return
	}

	// Check if we got any output
	if len(output) == 0 {
		sendErrorResponse(w, http.StatusInternalServerError, fmt.Sprintf("No output from Python script | stderr: %s", stderr.String()))
		return
	}

	// Ensure output is valid UTF-8
	outputStr := string(output)
	if !utf8.ValidString(outputStr) {
		outputStr = strings.ToValidUTF8(outputStr, "")
		output = []byte(outputStr)
	}

	// Parse Python script output
	var pyResponse PythonScriptResponse
	if err := json.Unmarshal(output, &pyResponse); err != nil {
		sendErrorResponse(w, http.StatusInternalServerError, fmt.Sprintf("Failed to decode Python output: %v | Raw output: %s", err, string(output)))
		return
	}

	// Handle Python script errors
	if !pyResponse.Success {
		errorMsg := "Python script reported failure"
		if pyResponse.Error != "" {
			errorMsg = pyResponse.Error
		}
		sendErrorResponse(w, http.StatusOK, errorMsg)
		return
	}

	// Transform Python response to our enhanced format
	response := BillCreationResponse{
		Success: true,
		Message: pyResponse.Message,
	}

	if pyResponse.Data != nil {
		response.Summary = &pyResponse.Data.Summary

		// Transform bills data and include PDF information
		for _, bill := range pyResponse.Data.Bills {
			enhancedBill := EnhancedBillData{
				BillID:        bill.BillID,
				ClientID:      bill.ClientID,
				TotalAmount:   bill.TotalAmount,
				Status:        bill.Status,
				Error:         bill.Error,
				ProductsCount: bill.ProductsCount,
				CreationDate:  bill.CreationDate,
			}

			// Add PDF information if available
			if bill.PDFContent != "" && bill.PDFFilename != "" {
				// Decode base64 to get actual size
				decodedSize := 0
				if decoded, err := base64.StdEncoding.DecodeString(bill.PDFContent); err == nil {
					decodedSize = len(decoded)
				}

				enhancedBill.PDFFile = &PDFFileInfo{
					Filename:    bill.PDFFilename,
					Content:     bill.PDFContent,
					Size:        decodedSize,
					ContentType: "application/pdf",
				}
			}

			response.Bills = append(response.Bills, enhancedBill)
		}
	}

	// Send successful response
	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(response); err != nil {
		log.Printf("Failed to encode response: %v", err)
	}
}

func sendErrorResponse(w http.ResponseWriter, statusCode int, errorMessage string) {
	resp := BillCreationResponse{
		Success: false,
		Error:   errorMessage,
	}
	w.WriteHeader(statusCode)
	json.NewEncoder(w).Encode(resp)
}
