// websocket_client.go
package yeastar

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/url"
	"strings"

	"github.com/gorilla/websocket"
)

func StartWebSocketClient(ctx context.Context, processor *EventProcessor) error {
	cfg := GlobalConfigManager.GetConfig()
	if cfg == nil {
		return fmt.Errorf("no config available")
	}

	token, err := GlobalTokenManager.WaitForToken(ctx)
	if err != nil {
		return fmt.Errorf("failed to get token: %w", err)
	}

	// Construct WebSocket URL
	baseURL := strings.TrimPrefix(cfg.ApiBaseUrl, "http://")
	baseURL = strings.TrimPrefix(baseURL, "https://")
	wsURL := url.URL{
		Scheme:   "wss",
		Host:     baseURL,
		Path:     "/openapi/v1.0/subscribe",
		RawQuery: fmt.Sprintf("access_token=%s", token.AccessToken),
	}

	log.Printf("Connecting to Yeastar WebSocket at %s", wsURL.String())

	conn, _, err := websocket.DefaultDialer.Dial(wsURL.String(), nil)
	if err != nil {
		return fmt.Errorf("failed to connect: %w", err)
	}
	defer conn.Close()

	// Subscribe to events
	topics := []int{30007, 30008, 30009, 30011, 30012, 30013, 30014, 30015, 30019, 30020, 30022, 30025, 30026, 30028, 30029}
	subMsg := map[string]interface{}{"topic_list": topics}
	if err := conn.WriteJSON(subMsg); err != nil {
		return fmt.Errorf("failed to subscribe: %w", err)
	}

	// Start reading messages
	for {
		select {
		case <-ctx.Done():
			return nil
		default:
			_, message, err := conn.ReadMessage()
			if err != nil {
				return fmt.Errorf("read error: %w", err)
			}

			var event map[string]interface{}
			if err := json.Unmarshal(message, &event); err != nil {
				log.Printf("Failed to unmarshal event: %v", err)
				continue
			}

			processor.Enqueue(event)
		}
	}
}
