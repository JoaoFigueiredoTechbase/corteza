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

type OnConnectCallback func()
type OnDisconnectCallback func(reason string)

// WebSocketService handles real-time event monitoring
type WebSocketService struct {
	configManager   *ConfigManager
	tokenManager    *TokenManager
	cortezaClient   *CortezaClient
	eventQueue      *EventQueue
	conn            *websocket.Conn
	mu              sync.RWMutex
	isConnected     bool
	stopChan        chan struct{}
	heartbeatTicker *time.Ticker
	heartbeatMu     sync.Mutex

	onConnect    OnConnectCallback
	onDisconnect OnDisconnectCallback
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

// SetOnConnect sets the callback function to be called when WebSocket connects
func (ws *WebSocketService) SetOnConnect(callback OnConnectCallback) {
	ws.onConnect = callback
}

// SetOnDisconnect sets the callback function to be called when WebSocket disconnects
func (ws *WebSocketService) SetOnDisconnect(callback OnDisconnectCallback) {
	ws.onDisconnect = callback
}

func NewWebSocketService(configManager *ConfigManager, tokenManager *TokenManager, cortezaClient *CortezaClient, eventQueue *EventQueue) *WebSocketService {
	log.Println("[WebSocketService] Initializing new instance with event queue")
	return &WebSocketService{
		configManager: configManager,
		tokenManager:  tokenManager,
		cortezaClient: cortezaClient,
		eventQueue:    eventQueue,
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

	if ws.onConnect != nil {
		ws.onConnect()
	}

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
	ws.heartbeatMu.Lock()
	defer ws.heartbeatMu.Unlock()

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

func (ws *WebSocketService) sendHeartbeat() error {
	ws.mu.RLock()
	defer ws.mu.RUnlock()

	if !ws.isConnected || ws.conn == nil {
		return fmt.Errorf("WebSocket not connected")
	}

	if err := ws.conn.WriteMessage(websocket.TextMessage, []byte("heartbeat")); err != nil {
		return fmt.Errorf("failed to send heartbeat: %w", err)
	}

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

	defer func() {
		ws.mu.Lock()
		wasConnected := ws.isConnected
		ws.isConnected = false
		ws.mu.Unlock()

		if wasConnected && ws.onDisconnect != nil {
			ws.onDisconnect("listen loop ended")
		}
	}()

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

			messageType, message, err := conn.ReadMessage()
			if err != nil {
				if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
					log.Printf("[WebSocketService] Unexpected close: %v", err)
				} else {
					log.Printf("[WebSocketService] Read error: %v", err)
				}
				return err
			}

			if messageType == websocket.TextMessage {
				messageStr := string(message)

				// Handle heartbeat responses
				if messageStr == "heartbeat response" {
					continue
				}

				// Handle heartbeat requests
				if messageStr == "heartbeat" {
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

				// Enqueue the event for processing by worker pool
				if ws.eventQueue != nil {
					if !ws.eventQueue.Enqueue(event) {
						log.Printf("[WebSocketService] WARNING: Event queue full, event dropped")
					}
				} else {
					log.Printf("[WebSocketService] ERROR: Event queue is nil")
				}
			}
		}
	}
}

func (ws *WebSocketService) Close() error {
	log.Println("[WebSocketService] Closing WebSocket connection")

	ws.mu.Lock()
	defer ws.mu.Unlock()

	ws.StopHeartbeat()

	wasConnected := ws.isConnected

	if ws.stopChan != nil {
		select {
		case <-ws.stopChan:
			// Channel already closed
		default:
			close(ws.stopChan)
		}
	}

	var closeErr error
	if ws.conn != nil {
		closeErr = ws.conn.Close()
		ws.conn = nil
		ws.isConnected = false
		log.Println("[WebSocketService] WebSocket connection closed")

		if wasConnected && ws.onDisconnect != nil {
			ws.onDisconnect("connection closed")
		}

		return closeErr
	}

	log.Println("[WebSocketService] No active WebSocket connection to close")
	return nil
}

func (ws *WebSocketService) IsConnected() bool {
	ws.mu.RLock()
	defer ws.mu.RUnlock()
	return ws.isConnected
}