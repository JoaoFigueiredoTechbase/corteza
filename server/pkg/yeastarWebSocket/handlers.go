package yeastar

import (
	"context"
	"encoding/json"
	"fmt"
)

func RegisterDefaultHandlers(ctx context.Context, processor *EventProcessor) {
	// Call status (30011)
	processor.AddHandler(30011, func(event map[string]interface{}) error {
		msg, ok := event["msg"].(string)
		if !ok {
			return fmt.Errorf("missing msg field"
		
		
		
		
		)
		}

		var data map[string]interface{}
		if err := json.Unmarshal([]byte(msg), &data); err != nil {
			return err
		}

		return processor.BroadcastEvent(ctx, "call.status", data)
	})

	// Call Detail Record (30012)
	processor.AddHandler(30012, func(event map[string]interface{}) error {
		return processor.BroadcastEvent(ctx, "cdr", event)
	})
}
