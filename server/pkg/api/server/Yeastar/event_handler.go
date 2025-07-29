package Yeastar

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"regexp"
	"strings"
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

// Create a separate function for cleaning if needed
func verifyMessageWithCleaning(event map[string]interface{}) (map[string]interface{}, error) {
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

	// Clean the message (equivalent to PHP's trim and preg_replace)
	cleanedMsg := cleanWhitespace(msgString)

	var msg map[string]interface{}
	if err := json.Unmarshal([]byte(cleanedMsg), &msg); err != nil {
		log.Printf("ERROR: Failed to decode msg JSON: %s, error: %v", cleanedMsg, err)
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

func sendEventToEndpoint(event interface{}, endpoint string) error {
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

func cleanWhitespace(input string) string {
	re := regexp.MustCompile(`\s+`)
	cleaned := re.ReplaceAllString(input, " ")
	return strings.TrimSpace(cleaned)
}

func processMembers(msg map[string]interface{}) []CallMember {
	var members []CallMember

	membersData, exists := msg["members"]
	if !exists {
		return members
	}

	membersArray, ok := membersData.([]interface{})
	if !ok {
		return members
	}

	for _, memberData := range membersArray {
		member, ok := memberData.(map[string]interface{})
		if !ok {
			continue
		}

		// Check for extension
		if extensionData, exists := member["extension"]; exists {
			if ext, ok := extensionData.(map[string]interface{}); ok {
				members = append(members, CallMember{
					Type:      "extension",
					Number:    getStringPointer(ext, "number"),
					ChannelID: getStringPointer(ext, "channel_id"),
					Status:    getStringPointer(ext, "member_status"),
					CallPath:  getStringPointer(ext, "call_path"),
				})
			}
		}

		// Check for inbound
		if inboundData, exists := member["inbound"]; exists {
			if inb, ok := inboundData.(map[string]interface{}); ok {
				members = append(members, CallMember{
					Type:      "inbound",
					From:      getStringPointer(inb, "from"),
					To:        getStringPointer(inb, "to"),
					TrunkName: getStringPointer(inb, "trunk_name"),
					ChannelID: getStringPointer(inb, "channel_id"),
					Status:    getStringPointer(inb, "member_status"),
					CallPath:  getStringPointer(inb, "call_path"),
				})
			}
		}

		// Check for outbound
		if outboundData, exists := member["outbound"]; exists {
			if out, ok := outboundData.(map[string]interface{}); ok {
				members = append(members, CallMember{
					Type:      "outbound",
					From:      getStringPointer(out, "from"),
					To:        getStringPointer(out, "to"),
					TrunkName: getStringPointer(out, "trunk_name"),
					ChannelID: getStringPointer(out, "channel_id"),
					Status:    getStringPointer(out, "member_status"),
					CallPath:  getStringPointer(out, "call_path"),
				})
			}
		}
	}

	return members
}

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
func handleEventExtensionCallStatus(event map[string]interface{}) (*ExtensionCallStatusEvent, error) {
	msg, err := verifyMessage(event)
	if err != nil {
		return nil, err
	}

	eventData := mapToExtensionCallStatus(event, msg)
	log.Printf("Successfully mapped EventExtensionCallStatus: %+v", eventData)

	endpoint := "https://your-api.com/events"
	if err := sendEventToEndpoint(eventData, endpoint); err != nil {
		log.Printf("Failed to send event to endpoint: %v", err)
		return nil, err
	}

	return eventData, nil
}

// 3009
func handleEventExtensionPresenceStatus(event map[string]interface{}) (*ExtensionPresenceStatusEvent, error) {
	msg, err := verifyMessage(event)
	if err != nil {
		return nil, err
	}

	eventData := mapToExtensionPresenceStatus(event, msg)
	log.Printf("Successfully mapped ExtensionPresenceStatus: %+v", eventData)

	endpoint := "https://your-api.com/events"
	if err := sendEventToEndpoint(eventData, endpoint); err != nil {
		log.Printf("Failed to send event to endpoint: %v", err)
		return nil, err
	}

	return eventData, nil
}

// 30011
func handleEventCallStatusChanged(event map[string]interface{}) (*CallStatusChangedEvent, error) {
	msg, err := verifyMessageWithCleaning(event)
	if err != nil {
		return nil, err
	}

	members := processMembers(msg)

	eventData := mapToEventCallStatusChanged(event, msg, members)
	log.Printf("Successfully mapped CallStatusChanged: %+v", eventData)

	endpoint := "https://your-api.com/events"
	if err := sendEventToEndpoint(eventData, endpoint); err != nil {
		log.Printf("Failed to send event to endpoint: %v", err)
		return nil, err
	}

	return eventData, nil
}

// 30012
func handleEventNewCDR(event map[string]interface{}) error {
	msg, err := verifyMessage(event)
	if err != nil {
		return nil, err
	}

	return nil
}

// 30013
func handleEventCallTransfer(event map[string]interface{}) error {
	msg, err := verifyMessage(event)
	if err != nil {
		return nil, err
	}

	return nil
}

// 30014
func handleEventCallFoward(event map[string]interface{}) error {
	msg, err := verifyMessage(event)
	if err != nil {
		return nil, err
	}

	return nil
}

// 30015
func handleEventCallStatus(event map[string]interface{}) error {
	msg, err := verifyMessage(event)
	if err != nil {
		return nil, err
	}

	return nil
}

// 30019
func handleEventSatisfaction(event map[string]interface{}) error {
	msg, err := verifyMessage(event)
	if err != nil {
		return nil, err
	}

	return nil
}

// 30020
func handleEventUaCSTACall(event map[string]interface{}) error {
	msg, err := verifyMessage(event)
	if err != nil {
		return nil, err
	}

	return nil
}

// 30022
func handleEventExtensionConfiguration(event map[string]interface{}) error {
	msg, err := verifyMessage(event)
	if err != nil {
		return nil, err
	}

	return nil
}

// 30025
func handleEventAgentPause(event map[string]interface{}) error {
	msg, err := verifyMessage(event)
	if err != nil {
		return nil, err
	}

	return nil
}

// 30026
func handleEventAgentRingTimeout(event map[string]interface{}) error {
	msg, err := verifyMessage(event)
	if err != nil {
		return nil, err
	}

	return nil
}

// //30027
// func handleEventReportDownload() error {

// }

// 30028
func handleEventCallNoteStatusChanged(event map[string]interface{}) error {
	msg, err := verifyMessage(event)
	if err != nil {
		return nil, err
	}

	return nil
}

// 30029
func handleEventAgentStatusChanged(event map[string]interface{}) error {
	msg, err := verifyMessage(event)
	if err != nil {
		return nil, err
	}

	return nil
}
