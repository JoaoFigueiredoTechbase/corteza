package PythonScrapper

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"
)

const (
	MaxScriptTimeout  = 60 * time.Minute
	MaxEmailLength    = 254
	MaxPasswordLength = 128
)

type Product struct {
	IdProduct        string  `json:"IdProduct"`
	Name             string  `json:"Name"`
	ShortName        string  `json:"ShortName"`
	TaxValue         float64 `json:"TaxValue"`
	IsService        bool    `json:"IsService"`
	HandlingType     string  `json:"HandlingType"`
	Price            float64 `json:"Price"`
	TaxIncluded      string  `json:"TaxIncluded"`
	Family           string  `json:"Family"`
	BrandName        string  `json:"BrandName"`
	BrandModels      string  `json:"BrandModels"`
	DirectDiscount   float64 `json:"DirectDiscount"`
	ShortDescription string  `json"ShortDescription"`
	LongDescription  string  `json"LongDescription"`
}

type Response struct {
	Success  bool      `json:"success"`
	Products []Product `json:"products,omitempty"`
	Error    string    `json:"error,omitempty"`
	Count    int       `json:"count,omitempty"`
}

type ScrapeRequest struct {
	Email string `json:"email"`
	Senha string `json:"senha"`
}

type PythonOutput struct {
	Success  bool                     `json:"success"`
	Products []map[string]interface{} `json:"products"`
	Error    string                   `json:"error,omitempty"`
}

// cleanText removes control characters and normalizes whitespace
func cleanText(input string) string {
	if input == "" {
		return ""
	}

	// Ensure valid UTF-8
	if !utf8.ValidString(input) {
		input = strings.ToValidUTF8(input, "")
	}

	// Remove control characters except tabs and newlines
	var result strings.Builder
	for _, r := range input {
		if unicode.IsControl(r) && r != '\t' && r != '\n' {
			continue
		}
		result.WriteRune(r)
	}

	// Normalize whitespace
	text := strings.TrimSpace(result.String())
	re := regexp.MustCompile(`\s+`)
	return re.ReplaceAllString(text, " ")
}

// parseFloat safely converts interface{} to float64
func parseFloat(value interface{}) float64 {
	switch v := value.(type) {
	case float64:
		return v
	case float32:
		return float64(v)
	case int:
		return float64(v)
	case int64:
		return float64(v)
	case string:
		// Clean the string first
		cleaned := cleanText(v)
		if cleaned == "" {
			return 0.0
		}

		// Extract numeric part
		re := regexp.MustCompile(`[\d,]+\.?\d*`)
		match := re.FindString(cleaned)
		if match == "" {
			return 0.0
		}

		// Remove commas and parse
		numStr := strings.ReplaceAll(match, ",", "")
		if parsed, err := strconv.ParseFloat(numStr, 64); err == nil {
			return parsed
		}
	}
	return 0.0
}

// parseBool safely converts interface{} to bool
func parseBool(value interface{}) bool {
	switch v := value.(type) {
	case bool:
		return v
	case string:
		cleaned := strings.ToLower(strings.TrimSpace(v))
		return cleaned == "true" || cleaned == "1" || cleaned == "yes" || cleaned == "on"
	case int, int64, float64:
		return parseFloat(v) != 0
	}
	return false
}

// parseString safely converts interface{} to string
func parseString(value interface{}) string {
	if value == nil {
		return ""
	}

	str := fmt.Sprintf("%v", value)
	return cleanText(str)
}

// validateEmail performs basic email validation
func validateEmail(email string) bool {
	if len(email) > MaxEmailLength {
		return false
	}

	re := regexp.MustCompile(`^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$`)
	return re.MatchString(email)
}

// validatePassword performs basic password validation
func validatePassword(password string) bool {
	return len(password) > 0 && len(password) <= MaxPasswordLength
}

// convertProduct safely converts a map to Product struct
func convertProduct(productMap map[string]interface{}) Product {
	return Product{
		IdProduct:        parseString(productMap["IdProduct"]),
		Name:             parseString(productMap["Name"]),
		ShortName:        parseString(productMap["ShortName"]),
		TaxValue:         parseFloat(productMap["TaxValue"]),
		IsService:        parseBool(productMap["IsService"]),
		HandlingType:     parseString(productMap["HandlingType"]),
		Price:            parseFloat(productMap["Price"]),
		TaxIncluded:      parseString(productMap["TaxIncluded"]),
		Family:           parseString(productMap["Family"]),
		BrandName:        parseString(productMap["BrandName"]),
		BrandModels:      parseString(productMap["BrandModels"]),
		DirectDiscount:   parseFloat(productMap["DirectDiscount"]),
		ShortDescription: parseString(productMap["ShortDescription"]),
		LongDescription:  parseString(productMap["LongDescription"]),
	}
}

// validateProduct checks if product has meaningful data
func validateProduct(product Product) bool {
	return strings.TrimSpace(product.IdProduct) != "" || strings.TrimSpace(product.Name) != ""
}

func HandleScrapeKeyInvoiceProducts(w http.ResponseWriter, r *http.Request) {
	log.Println("Received Scrape Key Invoice Products request")

	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("X-Content-Type-Options", "nosniff")

	if r.Method != http.MethodPost {
		resp := Response{
			Success: false,
			Error:   "Method not allowed; Use POST",
		}
		w.WriteHeader(http.StatusMethodNotAllowed)
		json.NewEncoder(w).Encode(resp)
		return
	}

	var req ScrapeRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		resp := Response{
			Success: false,
			Error:   fmt.Sprintf("Failed to parse request body: %v", err),
		}
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(resp)
		return
	}

	req.Email = cleanText(req.Email)
	req.Senha = cleanText(req.Senha)

	if req.Email == "" || req.Senha == "" {
		resp := Response{
			Success: false,
			Error:   "Email and senha are required and cannot be empty",
		}
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(resp)
		return
	}

	if !validateEmail(req.Email) {
		resp := Response{
			Success: false,
			Error:   "Invalid email format",
		}
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(resp)
		return
	}

	if !validatePassword(req.Senha) {
		resp := Response{
			Success: false,
			Error:   "Invalid password",
		}
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(resp)
		return
	}

	// Prepare script execution
	cwd, _ := os.Getwd()
	scriptPath := filepath.Join(cwd, "pkg", "api", "server", "PythonScrapper", "python", "sync-products.py")

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

	// Create context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), MaxScriptTimeout)
	defer cancel()

	// Execute Python script with timeout
	cmd := exec.CommandContext(ctx, "py", scriptPath, req.Email, req.Senha)

	// Set environment variables for better Python execution
	cmd.Env = append(os.Environ(),
		"PYTHONIOENCODING=utf-8",
		"PYTHONUNBUFFERED=1",
	)

	// Capture both stdout and stderr separately for debugging
	var stdout, stderr strings.Builder
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()

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

	// Parse Python script output
	var pyData PythonOutput
	if err := json.Unmarshal(output, &pyData); err != nil {
		resp = Response{
			Success: false,
			Error:   fmt.Sprintf("Failed to decode Python output: %v | Raw output: %s | stderr: %s", err, string(output), stderr.String()),
		}
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(resp)
		return
	}

	// Handle Python script errors
	if !pyData.Success {
		errorMsg := pyData.Error
		if errorMsg == "" {
			errorMsg = "Python script reported failure"
		}
		resp = Response{
			Success: false,
			Error:   errorMsg,
		}
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(resp)
		return
	}

	// Process and clean products
	var cleanProducts []Product
	for _, productMap := range pyData.Products {
		product := convertProduct(productMap)

		// Validate product before including
		if validateProduct(product) {
			cleanProducts = append(cleanProducts, product)
		}
	}

	// Prepare final response
	if len(cleanProducts) == 0 {
		resp = Response{
			Success:  true,
			Error:    "No valid products found",
			Products: []Product{},
			Count:    0,
		}
	} else {
		resp = Response{
			Success:  true,
			Products: cleanProducts,
			Count:    len(cleanProducts),
		}
	}

	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(resp); err != nil {
		log.Printf("Failed to encode response: %v", err)
	}
}
