// types.go - Common types and interfaces for better organization
package PythonScrapper

import (
	"context"
	"encoding/json"
	"time"
)

// Configuration constants
const (
	MaxScriptTimeout  = 60 * time.Minute
	MaxEmailLength    = 254
	MaxPasswordLength = 128
	DefaultTimeout    = 30 * time.Second
)

// Common interfaces for better abstraction
type ScriptExecutor interface {
	//Execute(ctx context.Context, args ...string) (*ScriptResult, error)
	Execute(ctx context.Context, scriptName string, args ...string) (*ScriptResult, error)
}

type ResponseWriter interface {
	WriteSuccess(data interface{}) error
	WriteError(statusCode int, message string) error
}

type Validator interface {
	Validate() error
}

// Common request/response structures
type BaseRequest struct {
	Email string `json:"email"`
	Senha string `json:"senha"`
}

func (r *BaseRequest) Validate() error {
	r.Email = cleanText(r.Email)
	r.Senha = cleanText(r.Senha)

	if r.Email == "" || r.Senha == "" {
		return NewValidationError("Email and senha are required and cannot be empty")
	}

	if !validateEmail(r.Email) {
		return NewValidationError("Invalid email format")
	}

	if !validatePassword(r.Senha) {
		return NewValidationError("Invalid password")
	}

	return nil
}

type BaseResponse struct {
	Success bool   `json:"success"`
	Message string `json:"message,omitempty"`
	Error   string `json:"error,omitempty"`
}

type ScriptResult struct {
	Success  bool            `json:"success"`
	Data     json.RawMessage `json:"data,omitempty"`
	Error    string          `json:"error,omitempty"`
	Message  string          `json:"message,omitempty"`
	ExitCode int             `json:"exit_code,omitempty"`
	StdErr   string          `json:"stderr,omitempty"`
	Duration time.Duration   `json:"duration,omitempty"`
}

// Enhanced error types for better error handling
type ValidationError struct {
	Message string
	Field   string
}

func (e ValidationError) Error() string {
	if e.Field != "" {
		return e.Field + ": " + e.Message
	}
	return e.Message
}

func NewValidationError(message string) ValidationError {
	return ValidationError{Message: message}
}

func NewFieldValidationError(field, message string) ValidationError {
	return ValidationError{Field: field, Message: message}
}

type ScriptError struct {
	ScriptName string
	ExitCode   int
	StdErr     string
	Message    string
}

func (e ScriptError) Error() string {
	return e.ScriptName + ": " + e.Message
}

func NewScriptError(scriptName string, exitCode int, stderr, message string) ScriptError {
	return ScriptError{
		ScriptName: scriptName,
		ExitCode:   exitCode,
		StdErr:     stderr,
		Message:    message,
	}
}

// Product related types (consolidated from multiple files)
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
	ShortDescription string  `json:"ShortDescription"`
	LongDescription  string  `json:"LongDescription"`
}

type ProductBill struct {
	Details   string `json:"Details,omitempty"`
	Discount  string `json:"Discount"`
	IdProduct string `json:"IdProduct"`
	Price     string `json:"Price"`
	Quantity  string `json:"Quantity"`
	Tax       string `json:"Tax"`
}

// Order and bill related types
type Order struct {
	Address        string        `json:"Address"`
	AvencaRecordID string        `json:"AvencaRecordID"`
	DocDate        string        `json:"DocDate"`
	IdClient       string        `json:"IdClient"`
	Products       []ProductBill `json:"Products"`
}

type BillResult struct {
	BillID         string       `json:"bill_id"`
	ClientID       string       `json:"client_id"`
	AvencaRecordID string       `json:"avenca_record_id"`
	TotalAmount    float64      `json:"total_amount"`
	Status         string       `json:"status"`
	Error          string       `json:"error,omitempty"`
	ProductsCount  int          `json:"products_count"`
	CreationDate   string       `json:"creation_date"`
	PDFFile        *PDFFileInfo `json:"pdf_file,omitempty"`
}

type PDFFileInfo struct {
	Filename       string `json:"filename"`
	Content        string `json:"content"`
	Size           int    `json:"size"`
	ContentType    string `json:"content_type"`
	AvencaRecordID string `json:"avenca_record_id"`
}

type ProcessingSummary struct {
	TotalOrdersProcessed int     `json:"total_orders_processed"`
	SuccessfulBills      int     `json:"successful_bills"`
	FailedBills          int     `json:"failed_bills"`
	TotalRevenue         float64 `json:"total_revenue"`
	ProcessingTime       string  `json:"processing_time"`
}

// Serial number related types
type SerialNumberEntry struct {
	ProductName   string                   `json:"product_name"`
	SerialNumbers []map[string]interface{} `json:"serial_numbers"`
}

// KV wrapper types (for parsing complex JSON structures)
type KV struct {
	Value KVValue `json:"@value"`
	Type  string  `json:"@type"`
}

type KVValue struct {
	Address        string `json:"Address"`
	DocDate        string `json:"DocDate"`
	IdClient       string `json:"IdClient"`
	Products       string `json:"Products"`
	AvencaRecordID string `json:"AvencaRecordID"`
}

// Generic KV for products
type GenericKV[T any] struct {
	Value T      `json:"@value"`
	Type  string `json:"@type"`
}
