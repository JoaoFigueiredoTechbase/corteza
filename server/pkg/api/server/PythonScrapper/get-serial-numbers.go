package PythonScrapper

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
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

func HandleGettingSerialNumbers(w http.ResponseWriter, r *http.Request) {
	log.Println("Received Getting Serial Numbers request")

	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	body, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "Error reading request body", http.StatusBadRequest)
		return
	}
	defer r.Body.Close()

	var bodyRequest GSNBodyRequest
	if err := json.Unmarshal(body, &bodyRequest); err != nil {
		http.Error(w, fmt.Sprintf("Error parsing JSON: %v", err), http.StatusBadRequest)
		return
	}

	pythonCommand, err := convertToGSNPythonCommand(bodyRequest)
	if err != nil {
		http.Error(w, fmt.Sprintf("Error converting data: %v", err), http.StatusInternalServerError)
		return
	}

	fmt.Printf("Processed request for email: %s\n", pythonCommand.Email)
	fmt.Printf("Number of products: %d\n", len(pythonCommand.Products))
	for i, product := range pythonCommand.Products {
		fmt.Printf("Product %d: %s\n", i+1, product.ProductName)
	}

	w.Header().Set("Content-Type", "application/json")
	response := map[string]interface{}{
		"status":  "success",
		"message": "Request processed successfully",
		"data":    pythonCommand,
	}

	json.NewEncoder(w).Encode(response)
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
