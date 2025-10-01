package nifinformation

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"strings"
	"time"
)

const (
	APIBaseURL     = "https://www.nif.pt"
	RequestTimeout = 30 * time.Second
	APIRateLimit   = time.Minute
)

// ClientHandler lida com as requisições HTTP para busca de informações de clientes.
type ClientHandler struct {
	service ClientService
}

// NewClientHandler cria uma nova instância de ClientHandler.
func NewClientHandler(service ClientService) *ClientHandler {
	return &ClientHandler{service: service}
}

func (h *ClientHandler) HandleClientInformationSearch(w http.ResponseWriter, r *http.Request) {

	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Minute)
	defer cancel()

	log.Println("Requisição de busca de informações de cliente recebida")

	if r.Method != http.MethodPost {
		http.Error(w, "Método não permitido", http.StatusMethodNotAllowed)
		return
	}

	var payload ClientInformation
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		log.Printf("ERROR: Falha ao decodificar corpo da requisição: %v", err)
		http.Error(w, "Payload JSON inválido", http.StatusBadRequest)
		return
	}

	if payload.ApiKey == "" {
		log.Println("ERROR: Chave de API ausente")
		http.Error(w, "Chave de API é obrigatória", http.StatusBadRequest)
		return
	}

	resp, err := h.service.SearchClientInformation(ctx, payload)
	if err != nil {
		log.Printf("ERROR: Falha na busca de informações do cliente: %v", err)
		if strings.Contains(err.Error(), "NIF inválido") || strings.Contains(err.Error(), "nome do cliente não contém caracteres válidos") || strings.Contains(err.Error(), "é necessário fornecer") {
			http.Error(w, err.Error(), http.StatusBadRequest)
		} else if strings.Contains(err.Error(), "nenhum registo encontrado") || strings.Contains(err.Error(), "nenhuma correspondência adequada") {
			http.Error(w, err.Error(), http.StatusNotFound)
		} else if strings.Contains(err.Error(), "cancelado durante a espera do limite de taxa") {
			http.Error(w, "Requisição expirou", http.StatusRequestTimeout)
		} else {
			http.Error(w, "Falha ao buscar informações do cliente", http.StatusInternalServerError)
		}
		return
	}

	log.Printf("INFO: Informações recuperadas com sucesso para NIF: %d, Créditos: %+v", resp.Data.Nif, resp.Credits)
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(resp); err != nil {
		log.Printf("ERROR: Falha ao codificar resposta: %v", err)
		http.Error(w, "Falha ao codificar resposta", http.StatusInternalServerError)
	}
}
