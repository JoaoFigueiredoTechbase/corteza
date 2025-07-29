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

func getIntPointer(data map[string]interface{}, key string) *int {
	if value, exists := data[key]; exists {
		if intVal, ok := value.(int); ok {
			return &intVal
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
