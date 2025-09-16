package PythonScrapper

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

type HTTPHandler struct {
	productService      ProductService
	billService         BillService
	serialNumberService SerialNumberService
	logger              Logger
	scriptsDir          string
}

func NewHTTPHandler(scriptsDir string, logger Logger) *HTTPHandler {
	executor := NewPythonScriptExecutor(scriptsDir, logger)

	return &HTTPHandler{
		productService:      NewKeyInvoiceProductService(executor, scriptsDir, logger),
		billService:         NewKeyInvoiceBillService(executor, scriptsDir, logger),
		serialNumberService: NewKeyInvoiceSerialNumberService(executor, scriptsDir, logger),
		logger:              logger,
		scriptsDir:          scriptsDir,
	}
}

// Global handler instance
var handler *HTTPHandler

// Initialize the handler - MUST be called before registering routes
func InitHTTPHandler(scriptsDir string, logger Logger) {
	if logger == nil {
		logger = DefaultLogger{}
	}
	handler = NewHTTPHandler(scriptsDir, logger)
}

// Global handler functions that delegate to the initialized handler
func HandleScrapeKeyInvoiceProducts(w http.ResponseWriter, r *http.Request) {
	if handler == nil {
		http.Error(w, "Handler not initialized - call InitHTTPHandler first", http.StatusInternalServerError)
		return
	}
	handler.HandleScrapeKeyInvoiceProducts(w, r)
}

func HandleBillCreation(w http.ResponseWriter, r *http.Request) {
	if handler == nil {
		http.Error(w, "Handler not initialized - call InitHTTPHandler first", http.StatusInternalServerError)
		return
	}
	handler.HandleBillCreation(w, r)
}

func HandleGettingSerialNumbers(w http.ResponseWriter, r *http.Request) {
	if handler == nil {
		http.Error(w, "Handler not initialized - call InitHTTPHandler first", http.StatusInternalServerError)
		return
	}
	handler.HandleGettingSerialNumbers(w, r)
}

// Bill creation handler
func (h *HTTPHandler) HandleBillCreation(w http.ResponseWriter, r *http.Request) {
	h.logger.Info("Received bill creation request")

	writer := NewHTTPResponseWriter(w, h.logger)

	defer r.Body.Close()

	body, err := io.ReadAll(r.Body)
	if err != nil {
		writer.WriteError(http.StatusBadRequest, fmt.Sprintf("Failed to read request body: %v", err))
		return
	}

	// Parse wrapper request
	var req struct {
		Email  string          `json:"email"`
		Senha  string          `json:"senha"`
		Avenca json.RawMessage `json:"avenca"`
	}

	if err := json.Unmarshal(body, &req); err != nil {
		writer.WriteError(http.StatusBadRequest, fmt.Sprintf("Invalid request: %v", err))
		return
	}

	// Validate credentials
	baseReq := BaseRequest{Email: req.Email, Senha: req.Senha}
	if err := baseReq.Validate(); err != nil {
		writer.WriteError(http.StatusBadRequest, err.Error())
		return
	}

	// Parse orders
	orders, err := ParseOrders(req.Avenca, h.logger)
	if err != nil {
		writer.WriteError(http.StatusBadRequest, fmt.Sprintf("Failed to parse orders: %v", err))
		return
	}

	h.logger.Debug("Parsed %d orders for email=%s", len(orders), req.Email)

	ctx, cancel := context.WithTimeout(context.Background(), MaxScriptTimeout)
	defer cancel()

	result, err := h.billService.CreateBills(ctx, req.Email, req.Senha, orders)
	if err != nil {
		if scriptErr, ok := err.(ScriptError); ok {
			writer.WriteError(http.StatusInternalServerError, scriptErr.Error())
			return
		}
		writer.WriteError(http.StatusInternalServerError, err.Error())
		return
	}

	response := struct {
		BaseResponse
		Summary *ProcessingSummary `json:"summary,omitempty"`
		Bills   []BillResult       `json:"bills,omitempty"`
	}{
		BaseResponse: BaseResponse{
			Success: true,
			Message: fmt.Sprintf("Successfully processed %d orders, created %d bills",
				result.Summary.TotalOrdersProcessed, result.Summary.SuccessfulBills),
		},
		Summary: result.Summary,
		Bills:   result.Bills,
	}

	writer.WriteSuccess(response)
}

// Serial number handler
func (h *HTTPHandler) HandleGettingSerialNumbers(w http.ResponseWriter, r *http.Request) {
	h.logger.Info("Received serial number request")

	writer := NewHTTPResponseWriter(w, h.logger)

	if r.Method != http.MethodPost {
		writer.WriteError(http.StatusMethodNotAllowed, "Method not allowed; Use POST")
		return
	}

	body, err := io.ReadAll(r.Body)
	if err != nil {
		writer.WriteError(http.StatusBadRequest, fmt.Sprintf("Error reading request body: %v", err))
		return
	}
	defer r.Body.Close()

	var bodyRequest struct {
		Email    string                    `json:"email"`
		Senha    string                    `json:"senha"`
		Products []GenericKV[ArticleValue] `json:"Products"`
	}

	if err := json.Unmarshal(body, &bodyRequest); err != nil {
		writer.WriteError(http.StatusBadRequest, fmt.Sprintf("Error parsing JSON: %v", err))
		return
	}

	// Validate credentials
	baseReq := BaseRequest{Email: bodyRequest.Email, Senha: bodyRequest.Senha}
	if err := baseReq.Validate(); err != nil {
		writer.WriteError(http.StatusBadRequest, err.Error())
		return
	}

	// Convert to ArticleValue slice
	var products []ArticleValue
	for _, kv := range bodyRequest.Products {
		products = append(products, kv.Value)
	}

	ctx, cancel := context.WithTimeout(context.Background(), MaxScriptTimeout)
	defer cancel()

	serialNumbers, err := h.serialNumberService.GetSerialNumbers(ctx, bodyRequest.Email, bodyRequest.Senha, products)
	if err != nil {
		if scriptErr, ok := err.(ScriptError); ok {
			writer.WriteError(http.StatusInternalServerError, scriptErr.Error())
			return
		}
		writer.WriteError(http.StatusInternalServerError, err.Error())
		return
	}

	response := struct {
		BaseResponse
		SerialNumbers []SerialNumberEntry `json:"serial_numbers,omitempty"`
		Count         int                 `json:"count,omitempty"`
	}{
		BaseResponse:  BaseResponse{Success: true},
		SerialNumbers: serialNumbers,
		Count:         len(serialNumbers),
	}

	if len(serialNumbers) == 0 {
		response.Message = "No serial numbers found"
	}

	writer.WriteSuccess(response)
}

// Product scraping handler
func (h *HTTPHandler) HandleScrapeKeyInvoiceProducts(w http.ResponseWriter, r *http.Request) {
	h.logger.Info("Received product scraping request")

	writer := NewHTTPResponseWriter(w, h.logger)

	if r.Method != http.MethodPost {
		writer.WriteError(http.StatusMethodNotAllowed, "Method not allowed; Use POST")
		return
	}

	var req BaseRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writer.WriteError(http.StatusBadRequest, fmt.Sprintf("Failed to parse request body: %v", err))
		return
	}

	if err := req.Validate(); err != nil {
		writer.WriteError(http.StatusBadRequest, err.Error())
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), MaxScriptTimeout)
	defer cancel()

	products, err := h.productService.ScrapeProducts(ctx, req.Email, req.Senha)
	if err != nil {
		if scriptErr, ok := err.(ScriptError); ok {
			writer.WriteError(http.StatusInternalServerError, scriptErr.Error())
			return
		}
		writer.WriteError(http.StatusInternalServerError, err.Error())
		return
	}

	response := struct {
		BaseResponse
		Products []Product `json:"products,omitempty"`
		Count    int       `json:"count,omitempty"`
	}{
		BaseResponse: BaseResponse{Success: true},
		Products:     products,
		Count:        len(products),
	}

	if len(products) == 0 {
		response.Message = "No valid products found"
	}

	writer.WriteSuccess(response)
}
