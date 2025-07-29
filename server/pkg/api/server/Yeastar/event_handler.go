package Yeastar

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
)

// utils
func verifyMessage(event map[string]interface{}) (map[string]interface{}, error) {
	eventMessage, exists := event["msg"]

	if !exists {
		eventJSON, _ := json.Marshal(event)
		log.Printf("ERROR: msg field missing in the event: %s", string(eventJSON))
		return nil, fmt.Errorf("msg field missing in event")
	}

	msgString, ok := eventMessage.(string)
	if !ok {
		log.Printf("ERROR: msg field is not a string: %+v", eventMessage)
		return nil, fmt.Errorf("msg field is not a string, got type %T", eventMessage)
	}

	var msg map[string]interface{}
	if err := json.Unmarshal([]byte(msgString), &msg); err != nil {
		log.Printf("ERROR: Failed to decode msg JSON: %s, error: %v", msgString, err)
		return nil, fmt.Errorf("failed to decode msg JSON: %w", err)
	}

	log.Printf("Successfully decoded msg: %+v", msg)
	return msg, nil
}

func getStringPointer(data map[string]interface{}, key string) *string {
	if value, exists := data[key]; exists {
		if str, ok := value.(string); ok {
			return &str
		}
	}
	return nil
}

func sendEventToEndpoint(event *ExtensionRegistrationEvent, endpoint string) error {
	jsonData, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("failed to marshal event to JSON: %w", err)
	}

	log.Printf("Sending JSON to endpoint: %s", string(jsonData))

	resp, err := http.Post(endpoint, "application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("failed to send HTTP request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("endpoint returned status: %d", resp.StatusCode)
	}

	return nil
}

// struct

// 30007
func handleEventExtensionRegistration(event map[string]interface{}) (*ExtensionRegistrationEvent, error) {
	msg, err := verifyMessage(event)
	if err != nil {
		return nil, err
	}

	eventData := mapToExtensionRegistration(event, msg)

	log.Printf("Successfully mapped ExtensionRegistration: %+v", eventData)

	endpoint := "https://your-api.com/events"
	if err := sendEventToEndpoint(eventData, endpoint); err != nil {
		log.Printf("Failed to send event to endpoint: %v", err)
		return nil, err
	}

	return eventData, nil
}

// 30008
func handleEventExtensionCallStatus(event map[string]interface{}) error {
	_, err := verifyMessage(event)
	if err != nil {
		return err
	}

	return nil
}

// 3009
func handleEventExtensionPresenceStatus(event map[string]interface{}) error {
	_, err := verifyMessage(event)
	if err != nil {
		return err
	}

	return nil
}

// 30011
func handleEventCallStatusChanged(event map[string]interface{}) error {
	_, err := verifyMessage(event)
	if err != nil {
		return err
	}

	return nil
}

// 30012
func handleEventNewCDR(event map[string]interface{}) error {
	_, err := verifyMessage(event)
	if err != nil {
		return err
	}

	return nil
}

// 30013
func handleEventCallTransfer(event map[string]interface{}) error {
	_, err := verifyMessage(event)
	if err != nil {
		return err
	}

	return nil
}

// 30014
func handleEventCallFoward(event map[string]interface{}) error {
	_, err := verifyMessage(event)
	if err != nil {
		return err
	}

	return nil
}

// 30015
func handleEventCallStatus(event map[string]interface{}) error {
	_, err := verifyMessage(event)
	if err != nil {
		return err
	}

	return nil
}

// 30019
func handleEventSatisfaction(event map[string]interface{}) error {
	_, err := verifyMessage(event)
	if err != nil {
		return err
	}

	return nil
}

// 30020
func handleEventUaCSTACall(event map[string]interface{}) error {
	_, err := verifyMessage(event)
	if err != nil {
		return err
	}

	return nil
}

// 30022
func handleEventExtensionConfiguration(event map[string]interface{}) error {
	_, err := verifyMessage(event)
	if err != nil {
		return err
	}

	return nil
}

// 30025
func handleEventAgentPause(event map[string]interface{}) error {
	_, err := verifyMessage(event)
	if err != nil {
		return err
	}

	return nil
}

// 30026
func handleEventAgentRingTimeout(event map[string]interface{}) error {
	_, err := verifyMessage(event)
	if err != nil {
		return err
	}

	return nil
}

// //30027
// func handleEventReportDownload() error {

// }

// 30028
func handleEventCallNoteStatusChanged(event map[string]interface{}) error {
	_, err := verifyMessage(event)
	if err != nil {
		return err
	}

	return nil
}

// 30029
func handleEventAgentStatusChanged(event map[string]interface{}) error {
	_, err := verifyMessage(event)
	if err != nil {
		return err
	}

	return nil
}
