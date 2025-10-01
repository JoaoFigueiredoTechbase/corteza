package nifinformation

import (
	"context"
	"fmt"
	"log"
	"net/url"
	"strings"
	"time"
)

// ClientService define a interface para a lógica de negócio de busca de clientes.
type ClientService interface {
	SearchClientInformation(ctx context.Context, payload ClientInformation) (*ApiResponseWithCredits, error)
}

type nifClientService struct {
	apiBaseURL   string
	apiRateLimit time.Duration
}

// NewNifClientService cria uma nova instância de ClientService.
func NewNifClientService(apiBaseURL string, apiRateLimit time.Duration) ClientService {
	return &nifClientService{
		apiBaseURL:   apiBaseURL,
		apiRateLimit: apiRateLimit,
	}
}

func (s *nifClientService) SearchClientInformation(ctx context.Context, payload ClientInformation) (*ApiResponseWithCredits, error) {
	payload.ClientNif = strings.TrimSpace(payload.ClientNif)
	payload.ClientName = strings.TrimSpace(payload.ClientName)

	// Prioridade para client_nif
	if payload.ClientNif != "" {
		if !validateNif(payload.ClientNif) {
			return nil, fmt.Errorf("NIF inválido: %s", payload.ClientNif)
		}
		nif, err := parseNif(payload.ClientNif)
		if err != nil {
			return nil, fmt.Errorf("erro ao converter NIF: %w", err)
		}
		return s.fetchByNif(ctx, nif, payload.ApiKey)
	} else if payload.ClientName != "" {
		query := sanitizeQuery(payload.ClientName)
		if query == "" {
			return nil, fmt.Errorf("o nome do cliente não contém caracteres válidos")
		}
		return s.fetchByName(ctx, query, payload.ClientName, payload.ApiKey)
	} else {
		return nil, fmt.Errorf("é necessário fornecer client_nif (9 dígitos) ou client_name")
	}
}

func (s *nifClientService) fetchByNif(ctx context.Context, nif int, apiKey string) (*ApiResponseWithCredits, error) {
	apiURL := fmt.Sprintf("%s/?json=1&q=%d&key=%s", s.apiBaseURL, nif, url.QueryEscape(apiKey))
	log.Printf("INFO: A fazer pedido API para NIF: %d", nif)

	apiResp, err := makeAPIRequest(ctx, apiURL)
	if err != nil {
		return nil, fmt.Errorf("falha no pedido API para NIF %d: %w", nif, err)
	}

	for _, raw := range apiResp.Records {
		record, err := parseRecord(raw)
		if err != nil {
			return nil, fmt.Errorf("falha ao analisar registo para NIF %d: %w", nif, err)
		}
		return &ApiResponseWithCredits{
			Data:    record,
			Credits: apiResp.Credits,
		}, nil
	}

	return nil, fmt.Errorf("nenhum registo encontrado para NIF %d", nif)
}

func (s *nifClientService) fetchByName(ctx context.Context, sanitizedName, originalName, apiKey string) (*ApiResponseWithCredits, error) {
	// Primeira requisição com o nome
	apiURL := fmt.Sprintf("%s/?json=1&q=%s&key=%s", s.apiBaseURL, url.QueryEscape(sanitizedName), url.QueryEscape(apiKey))
	log.Printf("INFO: A fazer pedido API com nome: %s", sanitizedName)

	apiResp, err := makeAPIRequest(ctx, apiURL)
	if err != nil {
		return nil, fmt.Errorf("falha no pedido API com nome %s: %w", sanitizedName, err)
	}

	var results []NifApiResponse
	for _, raw := range apiResp.Records {
		record, err := parseRecord(raw)
		if err != nil {
			log.Printf("WARN: Falha ao analisar registo: %v", err)
			continue
		}
		if record.Nif != 0 {
			results = append(results, record)
		}
	}

	if len(results) == 0 {
		return nil, fmt.Errorf("nenhum registo encontrado para o nome: %s", originalName)
	}

	bestMatch, ok := pickBestMatch(results, originalName)
	if !ok {
		return nil, fmt.Errorf("nenhuma correspondência adequada encontrada para o nome: %s", originalName)
	}

	log.Printf("INFO: Melhor correspondência encontrada - NIF: %d, Título: %s", bestMatch.Nif, bestMatch.Title)
	log.Printf("INFO: A aguardar %v devido ao limite de taxa da API antes de obter detalhes completos", s.apiRateLimit)

	select {
	case <-time.After(s.apiRateLimit):
		// Continuar após o atraso
	case <-ctx.Done():
		return nil, fmt.Errorf("pedido cancelado durante a espera do limite de taxa: %w", ctx.Err())
	}

	// Segunda requisição com o NIF da melhor correspondência
	return s.fetchByNif(ctx, bestMatch.Nif, apiKey)
}
