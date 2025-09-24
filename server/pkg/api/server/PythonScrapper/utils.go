// utils.go - Utility functions and services
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

// Logger interface for better testability
type Logger interface {
	Info(msg string, args ...interface{})
	Error(msg string, args ...interface{})
	Debug(msg string, args ...interface{})
	Warn(msg string, args ...interface{})
}

// DefaultLogger implements Logger interface
type DefaultLogger struct{}

func (l DefaultLogger) Info(msg string, args ...interface{}) {
	log.Printf("[INFO] "+msg, args...)
}

func (l DefaultLogger) Error(msg string, args ...interface{}) {
	log.Printf("[ERROR] "+msg, args...)
}

func (l DefaultLogger) Debug(msg string, args ...interface{}) {
	log.Printf("[DEBUG] "+msg, args...)
}

func (l DefaultLogger) Warn(msg string, args ...interface{}) {
	log.Printf("[WARN] "+msg, args...)
}

// HTTPResponseWriter wraps http.ResponseWriter with our interface
type HTTPResponseWriter struct {
	w      http.ResponseWriter
	logger Logger
}

func NewHTTPResponseWriter(w http.ResponseWriter, logger Logger) *HTTPResponseWriter {
	return &HTTPResponseWriter{w: w, logger: logger}
}

func (h *HTTPResponseWriter) WriteSuccess(data interface{}) error {
	h.w.Header().Set("Content-Type", "application/json")
	h.w.Header().Set("X-Content-Type-Options", "nosniff")
	h.w.WriteHeader(http.StatusOK)

	return json.NewEncoder(h.w).Encode(data)
}

func (h *HTTPResponseWriter) WriteError(statusCode int, message string) error {
	h.logger.Error("HTTP error response: %d - %s", statusCode, message)

	resp := BaseResponse{
		Success: false,
		Error:   message,
	}

	h.w.Header().Set("Content-Type", "application/json")
	h.w.Header().Set("X-Content-Type-Options", "nosniff")
	h.w.WriteHeader(statusCode)

	return json.NewEncoder(h.w).Encode(resp)
}

// PythonScriptExecutor implements ScriptExecutor interface
type PythonScriptExecutor struct {
	pythonPath string
	scriptsDir string
	logger     Logger
}

func NewPythonScriptExecutor(scriptsDir string, logger Logger) *PythonScriptExecutor {
	pythonPath := "python3"
	if _, err := exec.LookPath("py"); err == nil {
		pythonPath = "py"
	}

	return &PythonScriptExecutor{
		pythonPath: pythonPath,
		scriptsDir: scriptsDir,
		logger:     logger,
	}
}

func (p *PythonScriptExecutor) Execute(ctx context.Context, scriptName string, args ...string) (*ScriptResult, error) {
	startTime := time.Now()

	// Build script path
	scriptPath := filepath.Join(p.scriptsDir, scriptName)
	if _, err := os.Stat(scriptPath); os.IsNotExist(err) {
		return nil, fmt.Errorf("script not found: %s", scriptPath)
	}

	// Prepare command
	cmdArgs := append([]string{scriptPath}, args...)
	cmd := exec.CommandContext(ctx, p.pythonPath, cmdArgs...)

	// Set environment
	cmd.Env = append(os.Environ(),
		"PYTHONIOENCODING=utf-8",
		"PYTHONUNBUFFERED=1",
	)

	p.logger.Debug("Executing Python script: %s", p.pythonPath)

	// Execute and capture output
	stdout, stderr, err := p.executeWithOutput(cmd)
	duration := time.Since(startTime)

	result := &ScriptResult{
		Duration: duration,
		StdErr:   string(stderr),
	}

	// Handle timeout
	if ctx.Err() == context.DeadlineExceeded {
		result.Error = "Script execution timed out"
		return result, NewScriptError(scriptName, -1, string(stderr), "timeout")
	}

	// Handle execution errors
	if err != nil {
		result.Error = err.Error()
		result.ExitCode = p.getExitCode(err)
		return result, NewScriptError(scriptName, result.ExitCode, string(stderr), err.Error())
	}

	// Validate and clean output
	if len(stdout) == 0 {
		result.Error = "No output from script"
		return result, NewScriptError(scriptName, 0, string(stderr), "empty output")
	}

	// Ensure valid UTF-8
	outputStr := string(stdout)
	if !utf8.ValidString(outputStr) {
		outputStr = strings.ToValidUTF8(outputStr, "")
	}

	// Try to parse as JSON
	var jsonOutput map[string]interface{}
	if err := json.Unmarshal([]byte(outputStr), &jsonOutput); err != nil {
		result.Error = fmt.Sprintf("Invalid JSON output: %v", err)
		return result, NewScriptError(scriptName, 0, string(stderr), "invalid JSON")
	}

	// Check if script reported success
	if success, ok := jsonOutput["success"].(bool); ok {
		result.Success = success
		if !success {
			if errMsg, ok := jsonOutput["error"].(string); ok {
				result.Error = errMsg
			}
		}
	}

	result.Data = json.RawMessage(outputStr)
	p.logger.Debug("Script execution completed in %v", duration)

	return result, nil
}

func (p *PythonScriptExecutor) executeWithOutput(cmd *exec.Cmd) ([]byte, []byte, error) {
	var stdout, stderr strings.Builder
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()

	if stderr.Len() > 0 {
		p.logger.Debug("Script stderr: %s", stderr.String())
	}

	return []byte(stdout.String()), []byte(stderr.String()), err
}

func (p *PythonScriptExecutor) getExitCode(err error) int {
	if exitError, ok := err.(*exec.ExitError); ok {
		return exitError.ExitCode()
	}
	return -1
}

// Text processing utilities
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

// Number parsing utilities
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
		cleaned := cleanText(v)
		if cleaned == "" {
			return 0.0
		}

		re := regexp.MustCompile(`[\d,]+\.?\d*`)
		match := re.FindString(cleaned)
		if match == "" {
			return 0.0
		}

		numStr := strings.ReplaceAll(match, ",", "")
		if parsed, err := strconv.ParseFloat(numStr, 64); err == nil {
			return parsed
		}
	}
	return 0.0
}

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

func parseString(value interface{}) string {
	if value == nil {
		return ""
	}
	str := fmt.Sprintf("%v", value)
	return cleanText(str)
}

// Validation utilities
func validateEmail(email string) bool {
	if len(email) > MaxEmailLength {
		return false
	}
	re := regexp.MustCompile(`^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$`)
	return re.MatchString(email)
}

func validatePassword(password string) bool {
	return len(password) > 0 && len(password) <= MaxPasswordLength
}

// Product conversion utilities
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

func validateProduct(product Product) bool {
	return strings.TrimSpace(product.IdProduct) != "" || strings.TrimSpace(product.Name) != ""
}

// Order parsing utilities
func ParseOrders(data []byte, logger Logger) ([]Order, error) {
	logger.Debug("Parsing orders from JSON data: %d bytes", len(data))

	var kvs []KV
	if err := json.Unmarshal(data, &kvs); err != nil {
		return nil, fmt.Errorf("failed to unmarshal KV data: %w", err)
	}

	logger.Debug("Parsed %d KV entries", len(kvs))

	var orders []Order
	for i, kv := range kvs {
		logger.Debug("Processing KV %d - IdClient: %s, AvencaRecordID: %s",
			i, kv.Value.IdClient, kv.Value.AvencaRecordID)

		order := Order{
			Address:        kv.Value.Address,
			AvencaRecordID: kv.Value.AvencaRecordID,
			DocDate:        kv.Value.DocDate,
			IdClient:       kv.Value.IdClient,
		}

		// Parse products string
		productsStr := kv.Value.Products
		if strings.Contains(productsStr, "],[") {
			productsStr = strings.ReplaceAll(productsStr, "],[", ",")
			logger.Debug("Fixed products string format")
		}

		var rawProducts []struct {
			Value ProductBill `json:"@value"`
		}
		if err := json.Unmarshal([]byte(productsStr), &rawProducts); err != nil {
			logger.Error("Error parsing products for client %s: %v", kv.Value.IdClient, err)
			return nil, fmt.Errorf("error parsing products for client %s: %w", kv.Value.IdClient, err)
		}

		for _, p := range rawProducts {
			order.Products = append(order.Products, p.Value)
		}

		logger.Debug("Order has %d products", len(order.Products))
		orders = append(orders, order)
	}

	// Remove duplicate details
	orders = removeDuplicateProductDetails(orders, logger)
	logger.Debug("Returning %d orders", len(orders))

	return orders, nil
}

func removeDuplicateProductDetails(orders []Order, logger Logger) []Order {
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

	// Find which details appear multiple times
	detailsLastSeen := make(map[string]int)

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
				pp.Product.Details = ""
				logger.Debug("Cleared duplicate details '%s' from order %d, product %d",
					details, pp.OrderIndex, pp.ProductIndex)
			}
		}
	}

	return orders
}
