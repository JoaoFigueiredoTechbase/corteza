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

type GSNBodyRequest struct {
	Email    string  `json:"email"`
	Senha    string  `json:"senha"`
	Products []GSNKV `json:"Products"`
}

type GSNPythonCommand struct {
	Email    string         `json:"email"`
	Senha    string         `json:"senha"`
	Products []ArticleValue `json:"Products"`
}

type ArticleValue struct {
	ProductName string `json:"ProductName"`
}

type GSNKV struct {
	Value ArticleValue `json:"@value"`
	Type  string       `json:"@type"`
}

// Response structure for serial numbers
type GSNResponse struct {
	Success       bool                     `json:"success"`
	SerialNumbers []map[string]interface{} `json:"serial_numbers,omitempty"`
	Error         string                   `json:"error,omitempty"`
	Count         int                      `json:"count,omitempty"`
}

// Python output structure for serial numbers
type GSNPythonOutput struct {
	Success       bool                     `json:"success"`
	SerialNumbers []map[string]interface{} `json:"serial_numbers"`
	Error         string                   `json:"error,omitempty"`
}

func HandleGettingSerialNumbers(w http.ResponseWriter, r *http.Request) {
	log.Println("Received Getting Serial Numbers request")

	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("X-Content-Type-Options", "nosniff")

	if r.Method != http.MethodPost {
		resp := GSNResponse{
			Success: false,
			Error:   "Method not allowed; Use POST",
		}
		w.WriteHeader(http.StatusMethodNotAllowed)
		json.NewEncoder(w).Encode(resp)
		return
	}

	body, err := io.ReadAll(r.Body)
	if err != nil {
		resp := GSNResponse{
			Success: false,
			Error:   fmt.Sprintf("Error reading request body: %v", err),
		}
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(resp)
		return
	}
	defer r.Body.Close()

	var bodyRequest GSNBodyRequest
	if err := json.Unmarshal(body, &bodyRequest); err != nil {
		resp := GSNResponse{
			Success: false,
			Error:   fmt.Sprintf("Error parsing JSON: %v", err),
		}
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(resp)
		return
	}

	// Clean and validate input using your existing functions
	bodyRequest.Email = cleanText(bodyRequest.Email)
	bodyRequest.Senha = cleanText(bodyRequest.Senha)

	if bodyRequest.Email == "" || bodyRequest.Senha == "" {
		resp := GSNResponse{
			Success: false,
			Error:   "Email and senha are required and cannot be empty",
		}
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(resp)
		return
	}

	if !validateEmail(bodyRequest.Email) {
		resp := GSNResponse{
			Success: false,
			Error:   "Invalid email format",
		}
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(resp)
		return
	}

	if !validatePassword(bodyRequest.Senha) {
		resp := GSNResponse{
			Success: false,
			Error:   "Invalid password",
		}
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(resp)
		return
	}

	// Convert to Python command structure
	pythonCommand, err := convertToGSNPythonCommand(bodyRequest)
	if err != nil {
		resp := GSNResponse{
			Success: false,
			Error:   fmt.Sprintf("Error converting data: %v", err),
		}
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(resp)
		return
	}

	// Convert the command to JSON to pass to Python script
	commandJSON, err := json.Marshal(pythonCommand)
	if err != nil {
		resp := GSNResponse{
			Success: false,
			Error:   fmt.Sprintf("Error marshaling command to JSON: %v", err),
		}
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(resp)
		return
	}

	// Prepare script execution
	cwd, _ := os.Getwd()
	scriptPath := filepath.Join(cwd, "pkg", "api", "server", "PythonScrapper", "python", "get-serial-numbers.py")

	// Check if script exists
	if _, err := os.Stat(scriptPath); os.IsNotExist(err) {
		resp := GSNResponse{
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

	// Execute Python script with JSON as stdin or argument
	// Option 1: Pass JSON as argument (you might want to use stdin instead for large data)
	cmd := exec.CommandContext(ctx, "py", scriptPath, string(commandJSON))

	// Option 2: Pass JSON as stdin (uncomment this and comment above if you prefer)
	// cmd := exec.CommandContext(ctx, "py", scriptPath)
	// cmd.Stdin = strings.NewReader(string(commandJSON))

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

	// Log stderr for debugging
	if stderr.Len() > 0 {
		log.Printf("Python script stderr: %s", stderr.String())
	}

	output := []byte(stdout.String())

	var resp GSNResponse

	// Check for timeout
	if ctx.Err() == context.DeadlineExceeded {
		resp = GSNResponse{
			Success: false,
			Error:   "Script execution timed out",
		}
		w.WriteHeader(http.StatusRequestTimeout)
		json.NewEncoder(w).Encode(resp)
		return
	}

	// Check for execution errors
	if err != nil {
		errorMsg := fmt.Sprintf("Failed to execute Python script: %v", err)
		if stderr.Len() > 0 {
			errorMsg += fmt.Sprintf(" | stderr: %s", stderr.String())
		}

		resp = GSNResponse{
			Success: false,
			Error:   errorMsg,
		}
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(resp)
		return
	}

	// Check if we got any output
	if len(output) == 0 {
		resp = GSNResponse{
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
	var pyData GSNPythonOutput
	if err := json.Unmarshal(output, &pyData); err != nil {
		resp = GSNResponse{
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
		resp = GSNResponse{
			Success: false,
			Error:   errorMsg,
		}
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(resp)
		return
	}

	// Prepare final response
	if len(pyData.SerialNumbers) == 0 {
		resp = GSNResponse{
			Success:       true,
			Error:         "No serial numbers found",
			SerialNumbers: []map[string]interface{}{},
			Count:         0,
		}
	} else {
		resp = GSNResponse{
			Success:       true,
			SerialNumbers: pyData.SerialNumbers,
			Count:         len(pyData.SerialNumbers),
		}
	}

	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(resp); err != nil {
		log.Printf("Failed to encode response: %v", err)
	}
}

func convertToGSNPythonCommand(bodyRequest GSNBodyRequest) (GSNPythonCommand, error) {
	pythonCommand := GSNPythonCommand{
		Email: bodyRequest.Email,
		Senha: bodyRequest.Senha,
	}

	for _, kv := range bodyRequest.Products {
		articleValue := ArticleValue{
			ProductName: kv.Value.ProductName,
		}
		pythonCommand.Products = append(pythonCommand.Products, articleValue)
	}

	return pythonCommand, nil
}
