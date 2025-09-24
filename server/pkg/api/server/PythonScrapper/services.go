// services.go - Service layer for business logic
package PythonScrapper

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
)

// Service interfaces
type ProductService interface {
	ScrapeProducts(ctx context.Context, email, senha string) ([]Product, error)
}

type BillService interface {
	CreateBills(ctx context.Context, email, senha string, orders []Order) (*BillCreationResult, error)
}

type SerialNumberService interface {
	GetSerialNumbers(ctx context.Context, email, senha string, products []ArticleValue) ([]SerialNumberEntry, error)
}

// Service implementations
type KeyInvoiceProductService struct {
	executor   ScriptExecutor
	scriptsDir string
	logger     Logger
}

func NewKeyInvoiceProductService(executor ScriptExecutor, scriptsDir string, logger Logger) *KeyInvoiceProductService {
	return &KeyInvoiceProductService{
		executor:   executor,
		scriptsDir: scriptsDir,
		logger:     logger,
	}
}

func (s *KeyInvoiceProductService) ScrapeProducts(ctx context.Context, email, senha string) ([]Product, error) {
	s.logger.Info("Starting product scraping for email: %s", email)

	result, err := s.executor.Execute(ctx, "sync-products.py", email, senha)
	if err != nil {
		s.logger.Error("Product scraping failed: %v", err)
		return nil, err
	}

	if !result.Success {
		return nil, fmt.Errorf("product scraping script failed: %s", result.Error)
	}

	var response struct {
		Success  bool                     `json:"success"`
		Products []map[string]interface{} `json:"products"`
		Error    string                   `json:"error,omitempty"`
	}

	if err := json.Unmarshal(result.Data, &response); err != nil {
		return nil, fmt.Errorf("failed to parse product response: %w", err)
	}

	if !response.Success {
		return nil, fmt.Errorf("product scraping failed: %s", response.Error)
	}

	// Convert products
	var products []Product
	for _, productMap := range response.Products {
		product := convertProduct(productMap)
		if validateProduct(product) {
			products = append(products, product)
		}
	}

	s.logger.Info("Successfully scraped %d products", len(products))
	return products, nil
}

type KeyInvoiceBillService struct {
	executor   ScriptExecutor
	scriptsDir string
	logger     Logger
}

func NewKeyInvoiceBillService(executor ScriptExecutor, scriptsDir string, logger Logger) *KeyInvoiceBillService {
	return &KeyInvoiceBillService{
		executor:   executor,
		scriptsDir: scriptsDir,
		logger:     logger,
	}
}

type BillCreationResult struct {
	Summary  *ProcessingSummary `json:"summary"`
	Bills    []BillResult       `json:"bills"`
	PDFFiles []PDFFileInfo      `json:"pdf_files,omitempty"`
}

func (s *KeyInvoiceBillService) CreateBills(ctx context.Context, email, senha string, orders []Order) (*BillCreationResult, error) {
	s.logger.Info("Starting bill creation for %d orders", len(orders))

	ordersJSON, err := json.Marshal(orders)
	if err != nil {
		return nil, fmt.Errorf("failed to serialize orders: %w", err)
	}

	result, err := s.executor.Execute(ctx, "bill-creator.py", email, senha, string(ordersJSON))
	if err != nil {
		s.logger.Error("Bill creation failed: %v", err)
		return nil, err
	}

	if !result.Success {
		return nil, fmt.Errorf("bill creation script failed: %s", result.Error)
	}

	var response struct {
		Success bool                    `json:"success"`
		Data    *BillCreationScriptData `json:"data"`
		Error   string                  `json:"error,omitempty"`
	}

	if err := json.Unmarshal(result.Data, &response); err != nil {
		return nil, fmt.Errorf("failed to parse bill response: %w", err)
	}

	if !response.Success {
		return nil, fmt.Errorf("bill creation failed: %s", response.Error)
	}

	// Transform the response
	bills := make([]BillResult, len(response.Data.Bills))
	for i, bill := range response.Data.Bills {
		bills[i] = BillResult{
			BillID:         bill.BillID,
			ClientID:       bill.ClientID,
			AvencaRecordID: bill.AvencaRecordID,
			TotalAmount:    bill.TotalAmount,
			Status:         bill.Status,
			Error:          bill.Error,
			ProductsCount:  bill.ProductsCount,
			CreationDate:   bill.CreationDate,
		}

		// Handle PDF content
		if bill.PDFContent != "" && bill.PDFFilename != "" {
			decodedSize := 0
			if decoded, err := base64.StdEncoding.DecodeString(bill.PDFContent); err == nil {
				decodedSize = len(decoded)
			}

			bills[i].PDFFile = &PDFFileInfo{
				Filename:       bill.PDFFilename,
				Content:        bill.PDFContent,
				Size:           decodedSize,
				ContentType:    "application/pdf",
				AvencaRecordID: bill.AvencaRecordID,
			}
		}
	}

	resultObj := &BillCreationResult{
		Summary: &response.Data.Summary,
		Bills:   bills,
	}

	s.logger.Info("Successfully created %d bills", response.Data.Summary.SuccessfulBills)
	return resultObj, nil
}

// Additional types for bill service
type BillCreationScriptData struct {
	Summary  ProcessingSummary  `json:"summary"`
	Bills    []BillScriptResult `json:"bills"`
	PDFFiles []PDFFileInfo      `json:"pdf_files"`
}

type BillScriptResult struct {
	BillID         string  `json:"bill_id"`
	ClientID       string  `json:"client_id"`
	AvencaRecordID string  `json:"avenca_record_id"`
	TotalAmount    float64 `json:"total_amount"`
	Status         string  `json:"status"`
	Error          string  `json:"error,omitempty"`
	ProductsCount  int     `json:"products_count"`
	CreationDate   string  `json:"creation_date"`
	PDFFilename    string  `json:"pdf_filename"`
	PDFContent     string  `json:"pdf_content"`
}

type KeyInvoiceSerialNumberService struct {
	executor   ScriptExecutor
	scriptsDir string
	logger     Logger
}

func NewKeyInvoiceSerialNumberService(executor ScriptExecutor, scriptsDir string, logger Logger) *KeyInvoiceSerialNumberService {
	return &KeyInvoiceSerialNumberService{
		executor:   executor,
		scriptsDir: scriptsDir,
		logger:     logger,
	}
}

type ArticleValue struct {
	ProductName string `json:"ProductName"`
}

func (s *KeyInvoiceSerialNumberService) GetSerialNumbers(ctx context.Context, email, senha string, products []ArticleValue) ([]SerialNumberEntry, error) {
	s.logger.Info("Getting serial numbers for %d products", len(products))

	command := map[string]interface{}{
		"email":    email,
		"senha":    senha,
		"Products": products,
	}

	commandJSON, err := json.Marshal(command)
	if err != nil {
		return nil, fmt.Errorf("failed to serialize command: %w", err)
	}

	result, err := s.executor.Execute(ctx, "get-serial-numbers.py", string(commandJSON))
	if err != nil {
		s.logger.Error("Serial number retrieval failed: %v", err)
		return nil, err
	}

	if !result.Success {
		return nil, fmt.Errorf("serial number script failed: %s", result.Error)
	}

	var response struct {
		Success       bool                `json:"success"`
		SerialNumbers []SerialNumberEntry `json:"serial_numbers"`
		Error         string              `json:"error,omitempty"`
		Count         int                 `json:"count,omitempty"`
	}

	if err := json.Unmarshal(result.Data, &response); err != nil {
		return nil, fmt.Errorf("failed to parse serial number response: %w", err)
	}

	if !response.Success {
		return nil, fmt.Errorf("serial number retrieval failed: %s", response.Error)
	}

	s.logger.Info("Successfully retrieved serial numbers for %d products", response.Count)
	return response.SerialNumbers, nil
}
