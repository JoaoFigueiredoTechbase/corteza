package Yeastar

import (
	"bytes"
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

// WebSocketService handles real-time event monitoring
type WebSocketService struct {
	configManager   *ConfigManager
	tokenManager    *TokenManager
	cortezaClient   *CortezaClient
	conn            *websocket.Conn
	mu              sync.RWMutex
	isConnected     bool
	stopChan        chan struct{}
	heartbeatTicker *time.Ticker
}

type EventSubscription struct {
	TopicList []int `json:"topic_list"`
}

// EventResponse represents the response from subscription

type EventResponse struct {
	ErrCode int `json:"errcode"`

	ErrMsg string `json:"errmsg"`
}

// YeastarEvent represents an incoming event from Yeastar

type YeastarEvent struct {
	EventID int `json:"event_id"`

	EventType string `json:"event_type"`

	Timestamp int64 `json:"timestamp"`

	Data map[string]interface{} `json:"data"`
}

// Common event IDs from Yeastar API

const (
	EventExtensionRegistration   = 30007
	EventExtensionCallStatus     = 30008
	EventExtensionPresenceStatus = 30009
	EventCallStatusChanged       = 30011
	EventNewCDR                  = 30012
	EventCallTransfer            = 30013
	EventCallFoward              = 30014
	EventCallStatus              = 30015
	EventSatisfaction            = 30019
	EventUaCSTACall              = 30020
	EventExtensionConfiguration  = 30022
	EventAgentPause              = 30025
	EventAgentRingTimeout        = 30026
	EventReportDownload          = 30027
	EventCallNoteStatusChanged   = 30028
	EventAgentStatusChanged      = 30029
)

func NewWebSocketService(configManager *ConfigManager, tokenManager *TokenManager, cortezaClient *CortezaClient) *WebSocketService {
	log.Println("[WebSocketService] Initializing new instance")
	return &WebSocketService{
		configManager: configManager,
		tokenManager:  tokenManager,
		cortezaClient: cortezaClient,
		stopChan:      make(chan struct{}),
	}
}

func (ws *WebSocketService) Connect(ctx context.Context) error {
	log.Println("[WebSocketService] Attempting to connect...")

	config := ws.configManager.GetConfig()
	if config == nil {
		log.Println("[WebSocketService] No config available")
		return fmt.Errorf("config not available")
	}

	token := ws.tokenManager.GetToken()
	if token == nil || !ws.tokenManager.IsTokenValid() {
		log.Println("[WebSocketService] Invalid or missing token")
		return fmt.Errorf("valid token not available")
	}

	baseURL, err := url.Parse(config.ApiBaseUrl)
	if err != nil {
		log.Printf("[WebSocketService] Invalid API base URL: %v\n", err)
		return fmt.Errorf("invalid base URL: %w", err)
	}

	wsScheme := "ws"
	if baseURL.Scheme == "https" {
		wsScheme = "wss"
	}

	wsURL := fmt.Sprintf("%s://%s/openapi/v1.0/subscribe?access_token=%s",
		wsScheme, baseURL.Host, token.AccessToken)

	log.Printf("[WebSocketService] Connecting to WebSocket: %s\n", wsURL)

	dialer := websocket.Dialer{
		HandshakeTimeout: 30 * time.Second,
		TLSClientConfig: &tls.Config{
			InsecureSkipVerify: true,
		},
	}

	conn, _, err := dialer.DialContext(ctx, wsURL, http.Header{
		"User-Agent": []string{"Corteza-Yeastar-Integration"},
	})
	if err != nil {
		log.Printf("[WebSocketService] Failed to connect to WebSocket: %v\n", err)
		return fmt.Errorf("failed to connect to WebSocket: %w", err)
	}

	ws.mu.Lock()
	ws.conn = conn
	ws.isConnected = true
	ws.mu.Unlock()

	log.Println("[WebSocketService] WebSocket connection established successfully")
	return nil
}

func (ws *WebSocketService) Subscribe(eventIDs []int) error {
	ws.mu.RLock()
	conn := ws.conn
	isConnected := ws.isConnected
	ws.mu.RUnlock()

	if !isConnected || conn == nil {
		log.Println("[WebSocketService] Subscription failed: WebSocket not connected")
		return fmt.Errorf("WebSocket not connected")
	}

	log.Printf("[WebSocketService] Subscribing to events: %v\n", eventIDs)
	subscription := EventSubscription{TopicList: eventIDs}

	if err := conn.WriteJSON(subscription); err != nil {
		log.Printf("[WebSocketService] Failed to send subscription: %v\n", err)
		return fmt.Errorf("failed to send subscription: %w", err)
	}

	var response EventResponse
	if err := conn.ReadJSON(&response); err != nil {
		log.Printf("[WebSocketService] Failed to read subscription response: %v\n", err)
		return fmt.Errorf("failed to read subscription response: %w", err)
	}

	if response.ErrCode != 0 {
		log.Printf("[WebSocketService] Subscription error: %s\n", response.ErrMsg)
		return fmt.Errorf("subscription failed: %s", response.ErrMsg)
	}

	log.Printf("[WebSocketService] Successfully subscribed to events: %v\n", eventIDs)
	return nil
}

func (ws *WebSocketService) StartHeartbeat() {
	log.Println("[WebSocketService] Starting heartbeat mechanism...")
	ws.heartbeatTicker = time.NewTicker(50 * time.Second)

	go func() {
		for {
			select {
			case <-ws.heartbeatTicker.C:
				if err := ws.sendHeartbeat(); err != nil {
					log.Printf("[WebSocketService] Heartbeat failed: %v\n", err)
					return
				}
			case <-ws.stopChan:
				log.Println("[WebSocketService] Heartbeat stopped")
				return
			}
		}
	}()
}

func (ws *WebSocketService) sendHeartbeat() error {
	ws.mu.RLock()
	conn := ws.conn
	isConnected := ws.isConnected
	ws.mu.RUnlock()

	if !isConnected || conn == nil {
		return fmt.Errorf("WebSocket not connected")
	}

	log.Println("[WebSocketService] 💓 Sending heartbeat")
	if err := conn.WriteMessage(websocket.TextMessage, []byte("heartbeat")); err != nil {
		return fmt.Errorf("failed to send heartbeat: %w", err)
	}

	// Don't try to read the response here - it will be handled in the Listen loop
	log.Println("[WebSocketService] Heartbeat sent, response will be handled in listener")
	return nil
}

func (ws *WebSocketService) Listen(ctx context.Context) error {
	log.Println("[WebSocketService] Starting event listener...")

	ws.mu.RLock()
	conn := ws.conn
	isConnected := ws.isConnected
	ws.mu.RUnlock()

	if !isConnected || conn == nil {
		log.Println("[WebSocketService] Listen aborted: WebSocket not connected")
		return fmt.Errorf("WebSocket not connected")
	}

	for {
		select {
		case <-ctx.Done():
			log.Println("[WebSocketService] Context cancelled, stopping listener")
			return ctx.Err()
		case <-ws.stopChan:
			log.Println("[WebSocketService] Stop signal received, exiting listener")
			return nil
		default:
			conn.SetReadDeadline(time.Now().Add(70 * time.Second))

			// Read the raw message first
			messageType, message, err := conn.ReadMessage()
			if err != nil {
				if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
					log.Printf("[WebSocketService] Unexpected WebSocket closure: %v\n", err)
					return err
				}
				log.Printf("[WebSocketService] Error reading message: %v\n", err)
				continue
			}

			// Handle different message types
			if messageType == websocket.TextMessage {
				messageStr := string(message)
				log.Printf("[WebSocketService] Received text message: %s\n", messageStr)

				// Check if it's a heartbeat response
				if messageStr == "heartbeat response" {
					log.Println("[WebSocketService] ✅ Heartbeat response received")
					continue
				}

				// Check if it's a heartbeat request (some systems send this)
				if messageStr == "heartbeat" {
					log.Println("[WebSocketService] 💓 Heartbeat request received, sending response")
					if err := conn.WriteMessage(websocket.TextMessage, []byte("heartbeat response")); err != nil {
						log.Printf("[WebSocketService] Failed to send heartbeat response: %v\n", err)
					}
					continue
				}

				// Try to parse as JSON event
				var event map[string]interface{}
				if err := json.Unmarshal(message, &event); err != nil {
					log.Printf("[WebSocketService] Message is not valid JSON, ignoring: %s\n", messageStr)
					continue
				}

				// Process the JSON event
				if err := ws.processEvent(ctx, event); err != nil {
					log.Printf("[WebSocketService] Error processing event: %v\n", err)
				}
			} else {
				log.Printf("[WebSocketService] Received non-text message type: %d\n", messageType)
			}
		}
	}
}

func (ws *WebSocketService) processEvent(ctx context.Context, event map[string]interface{}) error {
	log.Printf("[WebSocketService] Received event: %+v\n", event)

	eventType, _ := event["event_type"].(string)
	eventID, _ := event["event_id"].(float64)

	log.Printf("[WebSocketService] Processing event - Type: %s, ID: %.0f\n", eventType, eventID)

	// if err := ws.cortezaClient.SendData(ctx, "events", event); err != nil {
	// 	log.Printf("[WebSocketService] Failed to send event to Corteza: %v\n", err)
	// }

	webhookURL := "https://webhook.site/f138fe58-2d58-4255-a3c3-9f92649e1339"
	if err := sendEventToWebhook(ctx, webhookURL, event); err != nil {
		log.Printf("[WebSocketService] ❌ Failed to send event to webhook: %v\n", err)
	}

	switch int(eventID) {
	case EventExtensionRegistration:
		log.Println("[WebSocketService] EventExtensionRegistration")
		_, err := handleEventExtensionRegistration(event)
		if err != nil {
			log.Printf("Failed to handle extension registration: %v", err)
			return err
		}
	case EventExtensionCallStatus:
		log.Println("[WebSocketService] EventExtensionCallStatus")
		handleEventExtensionCallStatus(event)
	case EventExtensionPresenceStatus:
		log.Println("[WebSocketService] EventExtensionPresenceStatus")
		handleEventExtensionPresenceStatus(event)
	case EventCallStatusChanged:
		log.Println("[WebSocketService] EventCallStatusChanged")
		handleEventCallStatusChanged(event)
	case EventNewCDR:
		log.Println("[WebSocketService] EventNewCDR")
		handleEventNewCDR(event)
	case EventCallTransfer:
		log.Println("[WebSocketService] EventCallTransfer")
		handleEventCallTransfer(event)
	case EventCallFoward:
		log.Println("[WebSocketService] EventCallFoward")
		handleEventCallFoward(event)
	case EventCallStatus:
		log.Println("[WebSocketService] EventCallStatus")
		handleEventCallStatus(event)
	case EventSatisfaction:
		log.Println("[WebSocketService] EventSatisfaction")
		handleEventSatisfaction(event)
	case EventUaCSTACall:
		log.Println("[WebSocketService] EventUaCSTACall")
		handleEventUaCSTACall(event)
	case EventExtensionConfiguration:
		log.Println("[WebSocketService] EventExtensionConfiguration")
		handleEventExtensionConfiguration(event)
	case EventAgentPause:
		log.Println("[WebSocketService] EventAgentPause")
		handleEventAgentPause(event)
	case EventAgentRingTimeout:
		log.Println("[WebSocketService] EventAgentRingTimeout")
		handleEventAgentRingTimeout(event)
	case EventCallNoteStatusChanged:
		log.Println("[WebSocketService] EventCallNoteStatusChanged")
		handleEventCallNoteStatusChanged(event)
	case EventAgentStatusChanged:
		log.Println("[WebSocketService] EventAgentStatusChanged")
		handleEventAgentStatusChanged(event)
	default:
		log.Printf("[WebSocketService] Unknown event ID received: %.0f\n", eventID)
	}

	return nil
}

// Add this helper function to the same file
func sendEventToWebhook(ctx context.Context, webhookURL string, event map[string]interface{}) error {
	// Create enriched payload
	payload := map[string]interface{}{
		"timestamp":  time.Now().Unix(),
		"source":     "yeastar-integration",
		"event_data": event,
	}

	jsonData, err := json.MarshalIndent(payload, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal event: %w", err)
	}

	log.Printf("[WebhookService] Sending to webhook: %s\n", webhookURL)
	log.Printf("[WebhookService] Payload:\n%s\n", string(jsonData))

	client := &http.Client{Timeout: 30 * time.Second}
	req, err := http.NewRequestWithContext(ctx, "POST", webhookURL, bytes.NewReader(jsonData))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", "Yeastar-Integration/1.0")

	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("webhook request failed: %w", err)
	}
	defer resp.Body.Close()

	log.Printf("[WebhookService] Response status: %d\n", resp.StatusCode)

	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		log.Println("[WebhookService] ✅ Event sent successfully")
		return nil
	}

	return fmt.Errorf("webhook returned status %d", resp.StatusCode)
}

func (ws *WebSocketService) Close() error {
	log.Println("[WebSocketService] Closing WebSocket connection")

	ws.mu.Lock()
	defer ws.mu.Unlock()

	if ws.heartbeatTicker != nil {
		ws.heartbeatTicker.Stop()
	}

	close(ws.stopChan)

	if ws.conn != nil {
		err := ws.conn.Close()
		ws.conn = nil
		ws.isConnected = false
		log.Println("[WebSocketService] WebSocket connection closed")
		return err
	}

	log.Println("[WebSocketService] No active WebSocket connection to close")
	return nil
}

func (ws *WebSocketService) IsConnected() bool {
	ws.mu.RLock()
	defer ws.mu.RUnlock()
	return ws.isConnected
}
