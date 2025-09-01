package PythonScrapper

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
)

type Product struct {
	IdProduct      string `json:"IdProduct"`
	Name           string `json:"Name"`
	ShortName      string `json:"ShortName"`
	TaxValue       string `json:"TaxValue"`
	IsService      bool   `json:"IsService"`
	HandlingType   string `json:"HandlingType"`
	Price          string `json:"Price"`
	TaxIncluded    string `json:"TaxIncluded"`
	Family         string `json:"Family"`
	BrandName      string `json:"BrandName"`
	BrandModels    string `json:"BrandModels"`
	DirectDiscount string `json:"DirectDiscount"`
}

type Response struct {
	Success  bool      `json:"success"`
	Products []Product `json:"products,omitempty"`
	Error    string    `json:"error,omitempty"`
	Output   string    `json:"output,omitempty"` // Optional, for debugging
}

type ScrapeRequest struct {
	Email string `json:"email"`
	Senha string `json:"senha"`
}

func HandleScrapeKeyInvoiceProducts(w http.ResponseWriter, r *http.Request) {
	log.Println("Received Scrape Key Invoice Products request")

	if r.Method != http.MethodPost {
		resp := Response{
			Success: false,
			Error:   "Method not allowed; Use POST",
		}

		w.Header().Set("Content-Type", "application/json")
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
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(resp)
		return
	}

	// Validate email and senha
	if req.Email == "" || req.Senha == "" {
		resp := Response{
			Success: false,
			Error:   "Email and senha are required",
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(resp)
		return
	}

	cwd, _ := os.Getwd()
	scriptPath := filepath.Join(cwd, "pkg", "api", "server", "PythonScrapper", "python", "sync-products.py")

	cmd := exec.Command("py", scriptPath, req.Email, req.Senha) // use "py" on Windows locally
	output, err := cmd.CombinedOutput()

	type PythonOutput struct {
		Success  bool      `json:"success"`
		Products []Product `json:"products"`
		Error    string    `json:"error,omitempty"` // Optional: for Python script errors
	}

	var resp Response
	w.Header().Set("Content-Type", "application/json")

	if err != nil {
		resp = Response{
			Success: false,
			Error:   fmt.Sprintf("Failed to execute Python script: %v", err),
			// Output:  string(output),
		}
		w.WriteHeader(http.StatusInternalServerError)
		if encodeErr := json.NewEncoder(w).Encode(resp); encodeErr != nil {
			log.Println("Failed to write JSON response:", encodeErr)
		}
		return
	}

	// Decode the Python script's output
	var pyData PythonOutput
	if decodeErr := json.Unmarshal(output, &pyData); decodeErr != nil {
		resp = Response{
			Success: false,
			Error:   fmt.Sprintf("Failed to decode Python output: %v", decodeErr),
			// Output:  string(output),
		}
		w.WriteHeader(http.StatusInternalServerError)
		if encodeErr := json.NewEncoder(w).Encode(resp); encodeErr != nil {
			log.Println("Failed to write JSON response:", encodeErr)
		}
		return
	}

	// Check if the Python script reported failure
	if !pyData.Success {
		resp = Response{
			Success: false,
			Error:   pyData.Error, // Use error message from Python script if available
			// Output:  string(output),
		}
		w.WriteHeader(http.StatusOK) // Or http.StatusBadRequest if appropriate
		if encodeErr := json.NewEncoder(w).Encode(resp); encodeErr != nil {
			log.Println("Failed to write JSON response:", encodeErr)
		}
		return
	}

	// Validate products
	if len(pyData.Products) == 0 {
		resp = Response{
			Success: true,
			Error:   "No products found",
			// Output:  string(output),
		}
	} else {
		resp = Response{
			Success:  true,
			Products: pyData.Products,
			// Output:   string(output), // Optional: remove in production
		}
	}

	w.WriteHeader(http.StatusOK)
	if encodeErr := json.NewEncoder(w).Encode(resp); encodeErr != nil {
		log.Println("Failed to write JSON response:", encodeErr)
	}
}
