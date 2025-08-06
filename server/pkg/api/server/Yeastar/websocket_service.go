package Yeastar

import (
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
	heartbeatMu     sync.Mutex
}

type EventSubscription struct {
	TopicList []int `json:"topic_list"`
}

type EventResponse struct {
	ErrCode int    `json:"errcode"`
	ErrMsg  string `json:"errmsg"`
}

type YeastarEvent struct {
	EventID   int                    `json:"event_id"`
	EventType string                 `json:"event_type"`
	Timestamp int64                  `json:"timestamp"`
	Data      map[string]interface{} `json:"data"`
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
	EventCallFailed              = 30015
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
	ws.stopChan = make(chan struct{})
	ws.mu.Unlock()

	conn.SetPingHandler(func(string) error {
		conn.WriteControl(websocket.PongMessage, []byte{}, time.Now().Add(5*time.Second))
		return nil
	})

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

// Revert to the simpler, working heartbeat mechanism from the old code
func (ws *WebSocketService) StartHeartbeat() {
	ws.heartbeatMu.Lock()
	defer ws.heartbeatMu.Unlock()

	// Stop existing heartbeat if running
	if ws.heartbeatTicker != nil {
		ws.heartbeatTicker.Stop()
	}

	log.Println("[WebSocketService] Starting heartbeat mechanism...")
	ws.heartbeatTicker = time.NewTicker(50 * time.Second)

	go func() {
		defer func() {
			log.Println("[WebSocketService] Heartbeat goroutine exiting")
		}()

		for {
			select {
			case <-ws.heartbeatTicker.C:
				if err := ws.sendHeartbeat(); err != nil {
					log.Printf("[WebSocketService] Heartbeat failed: %v", err)
					return
				}
			case <-ws.stopChan:
				log.Println("[WebSocketService] Heartbeat stopped")
				return
			}
		}
	}()
}

func (ws *WebSocketService) StopHeartbeat() {
	ws.heartbeatMu.Lock()
	defer ws.heartbeatMu.Unlock()

	if ws.heartbeatTicker != nil {
		ws.heartbeatTicker.Stop()
		ws.heartbeatTicker = nil
	}
}

// Use the working heartbeat format from old code
func (ws *WebSocketService) sendHeartbeat() error {
	ws.mu.RLock()
	defer ws.mu.RUnlock()

	if !ws.isConnected || ws.conn == nil {
		return fmt.Errorf("WebSocket not connected")
	}

	log.Println("[WebSocketService] 💓 Sending heartbeat")
	// Use plain text heartbeat like the old working code
	if err := ws.conn.WriteMessage(websocket.TextMessage, []byte("heartbeat")); err != nil {
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
					log.Printf("[WebSocketService] Unexpected close: %v", err)
				} else {
					log.Printf("[WebSocketService] Read error: %v", err)
				}

				return err
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
				if err := ws.processEvent(event); err != nil {
					log.Printf("[WebSocketService] Error processing event: %v\n", err)
				}
			} else {
				log.Printf("[WebSocketService] Received non-text message type: %d\n", messageType)
			}
		}
	}
}

func (ws *WebSocketService) processEvent(event map[string]interface{}) error {
	log.Printf("[WebSocketService] Received event: %+v\n", event)

	// Get the correct field names from the nested event_data
	eventID, _ := event["type"].(float64) // "type" field contains the event ID
	sn, _ := event["sn"].(string)         // "sn" field

	log.Printf("[WebSocketService] Processing event - SN: %s, ID: %.0f\n", sn, eventID)

	// Skip if eventID is 0 - this indicates heartbeat or invalid event
	if eventID == 0 {
		log.Printf("[WebSocketService] Received event with ID 0, skipping (likely heartbeat acknowledgment)")
		return nil
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
		_, err := handleEventExtensionCallStatus(event)
		if err != nil {
			log.Printf("Failed to handle extension call status: %v", err)
			return err
		}
	case EventExtensionPresenceStatus:
		log.Println("[WebSocketService] EventExtensionPresenceStatus")
		_, err := handleEventExtensionPresenceStatus(event)
		if err != nil {
			log.Printf("Failed to handle extension presence status: %v", err)
			return err
		}
	case EventCallStatusChanged:
		log.Println("[WebSocketService] EventCallStatusChanged")
		_, err := handleEventCallStatusChanged(event)
		if err != nil {
			log.Printf("Failed to handle call status changed: %v", err)
			return err
		}
	case EventNewCDR:
		log.Println("[WebSocketService] EventNewCDR")
		_, err := handleEventNewCDR(event)
		if err != nil {
			log.Printf("Failed to handle new call: %v", err)
			return err
		}
	case EventCallTransfer:
		log.Println("[WebSocketService] EventCallTransfer")
		_, err := handleEventCallTransfer(event)
		if err != nil {
			log.Printf("Failed to handle call transfer: %v", err)
			return err
		}
	case EventCallFoward:
		log.Println("[WebSocketService] EventCallFoward")
		_, err := handleEventCallFoward(event)
		if err != nil {
			log.Printf("Failed to handle call foward: %v", err)
			return err
		}
	case EventCallFailed:
		log.Println("[WebSocketService] EventCallFailed")
		_, err := handleEventCallFailedStatus(event)
		if err != nil {
			log.Printf("Failed to handle call failed: %v", err)
			return err
		}
	case EventSatisfaction:
		log.Println("[WebSocketService] EventSatisfaction")
		_, err := handleEventSatisfaction(event)
		if err != nil {
			log.Printf("Failed to handle satisfaction status: %v", err)
			return err
		}
	case EventExtensionConfiguration:
		log.Println("[WebSocketService] EventExtensionConfiguration")
		_, err := handleEventExtensionConfiguration(event)
		if err != nil {
			log.Printf("Failed to handle extension configuration status: %v", err)
			return err
		}
	case EventAgentPause:
		log.Println("[WebSocketService] EventAgentPause")
		_, err := handleEventAgentPause(event)
		if err != nil {
			log.Printf("Failed to handle extension pause status: %v", err)
			return err
		}
	case EventAgentRingTimeout:
		log.Println("[WebSocketService] EventAgentRingTimeout")
		_, err := handleEventAgentRingTimeout(event)
		if err != nil {
			log.Printf("Failed to handle extension timeout status: %v", err)
			return err
		}
	case EventCallNoteStatusChanged:
		log.Println("[WebSocketService] EventCallNoteStatusChanged")
		_, err := handleEventCallNoteStatusChanged(event)
		if err != nil {
			log.Printf("Failed to handle call note status: %v", err)
			return err
		}
	case EventAgentStatusChanged:
		log.Println("[WebSocketService] EventAgentStatusChanged")
		_, err := handleEventAgentStatusChanged(event)
		if err != nil {
			log.Printf("Failed to handle extension change status: %v", err)
			return err
		}
	default:
		log.Printf("[WebSocketService] Unknown event ID received: %.0f\n", eventID)
	}

	return nil
}

func (ws *WebSocketService) Close() error {
	log.Println("[WebSocketService] Closing WebSocket connection")

	ws.mu.Lock()
	defer ws.mu.Unlock()

	// Stop heartbeat first
	ws.StopHeartbeat()

	// Close stop channel if it exists and isn't already closed
	if ws.stopChan != nil {
		select {
		case <-ws.stopChan:
			// Channel already closed
		default:
			close(ws.stopChan)
		}
	}

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
