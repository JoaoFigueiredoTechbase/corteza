package Yeastar

import (
	"encoding/json"
	"fmt"
	"log"
	"strconv"
	"strings"
)

func safeStringConvert(value interface{}) string {
	if value == nil {
		return ""
	}
	switch v := value.(type) {
	case string:
		return strings.TrimSpace(v)
	case float64:
		return strconv.FormatFloat(v, 'f', -1, 64)
	case int:
		return strconv.Itoa(v)
	case int64:
		return strconv.FormatInt(v, 10)
	case bool:
		return strconv.FormatBool(v)
	default:
		return fmt.Sprintf("%v", v)
	}
}

func safeIntConvert(value interface{}) int {
	if value == nil {
		return 0
	}
	switch v := value.(type) {
	case int:
		return v
	case int64:
		return int(v)
	case float64:
		return int(v)
	case string:
		if i, err := strconv.Atoi(strings.TrimSpace(v)); err == nil {
			return i
		}
		return 0
	default:
		return 0
	}
}

func safeInt64Convert(value interface{}) int64 {
	if value == nil {
		return 0
	}
	switch v := value.(type) {
	case int64:
		return v
	case int:
		return int64(v)
	case float64:
		return int64(v)
	case string:
		if i, err := strconv.ParseInt(strings.TrimSpace(v), 10, 64); err == nil {
			return i
		}
		return 0
	default:
		return 0
	}
}

func extractCallNoteInfo(callNote interface{}) (noteText, noteType, noteDescription string) {
	if callNote == nil {
		return "", "", ""
	}

	switch v := callNote.(type) {
	case string:
		// Plain string note
		return v, "", ""
	case map[string]interface{}:
		// New format with fields

		// Extract remark
		if remark, ok := v["remark"].(string); ok {
			noteDescription = strings.TrimSpace(remark)
		}

		// Extract first disposition_code_list item
		if dcl, ok := v["disposition_code_list"].([]interface{}); ok && len(dcl) > 0 {
			if first, ok := dcl[0].(map[string]interface{}); ok {
				if name, ok := first["name"].(string); ok {
					noteType = strings.TrimSpace(name)
				}
			}
		}

		noteText = fmt.Sprintf("Type: %s, Description: %s", noteType, noteDescription)
		return noteText, noteType, noteDescription

	default:
		return "", "", ""
	}
}

func mapAgent(raw Agent) Agent {
	agent := Agent{
		ID:       raw.ID,
		Presence: safeStringConvert(raw.Presence), // Use safeStringConvert for consistency, though raw.Presence is already string
		Number:   safeStringConvert(raw.Number),
		Name:     safeStringConvert(raw.Name),
		Email:    safeStringConvert(raw.Email),
	}

	// Set default values for empty fields
	if agent.Presence == "" {
		agent.Presence = "unknown"
	}
	if agent.Number == "" {
		agent.Number = "0000"
	}
	if agent.Name == "" {
		agent.Name = "Unknown Agent"
	}
	if agent.Email == "" {
		agent.Email = "" // Explicitly keep it as empty string if missing
	}

	return agent
}

func mapQueue(raw Queue) Queue {
	queue := Queue{
		Name:         safeStringConvert(raw.Name),
		Number:       safeStringConvert(raw.Number),
		RingStrategy: safeStringConvert(raw.RingStrategy),
		SLATime:      safeIntConvert(raw.SLATime),
	}

	// Set default values for empty fields
	if queue.Name == "" {
		queue.Name = "Unknown Queue"
	}
	if queue.Number == "" {
		queue.Number = "0000"
	}
	if queue.RingStrategy == "" {
		queue.RingStrategy = "ring_all"
	}
	if queue.SLATime == 0 {
		queue.SLATime = 60 // Default to 60 seconds
	}

	return queue
}
func mapCDR(raw CDR) CDR {
	noteText, noteType, noteDescription := extractCallNoteInfo(raw.CallNote)

	cdr := CDR{
		ID:                  raw.ID,
		Time:                safeStringConvert(raw.Time),
		CallFrom:            safeStringConvert(raw.CallFrom),
		CallTo:              safeStringConvert(raw.CallTo),
		Timestamp:           safeInt64Convert(raw.Timestamp),
		UID:                 safeStringConvert(raw.UID),
		SrcAddr:             safeStringConvert(raw.SrcAddr),
		Duration:            safeIntConvert(raw.Duration),
		RingDuration:        safeIntConvert(raw.RingDuration),
		TalkDuration:        safeIntConvert(raw.TalkDuration),
		Disposition:         safeStringConvert(raw.Disposition),
		CallType:            safeStringConvert(raw.CallType),
		Reason:              safeStringConvert(raw.Reason),
		CallFromNumber:      safeStringConvert(raw.CallFromNumber),
		CallFromName:        safeStringConvert(raw.CallFromName),
		CallToNumber:        safeStringConvert(raw.CallToNumber),
		CallToName:          safeStringConvert(raw.CallToName),
		CallID:              safeStringConvert(raw.CallID),
		CallNote:            noteText,
		CallNoteType:        noteType,
		CallNoteDescription: noteDescription,
		CallNoteID:          safeStringConvert(raw.CallNoteID),
		EnbCallNote:         safeIntConvert(raw.EnbCallNote),
		DID:                 safeStringConvert(raw.DID),
		DIDName:             safeStringConvert(raw.DIDName),
	}

	// Set default values for empty fields
	if cdr.Time == "" {
		cdr.Time = "1970-01-01 00:00:00"
	}
	if cdr.CallFrom == "" {
		cdr.CallFrom = "unknown"
	}
	if cdr.CallTo == "" {
		cdr.CallTo = "unknown"
	}
	if cdr.UID == "" {
		cdr.UID = "unknown"
	}
	if cdr.Disposition == "" {
		cdr.Disposition = "UNKNOWN"
	}
	if cdr.CallType == "" {
		cdr.CallType = "unknown"
	}
	if cdr.Reason == "" {
		cdr.Reason = "unknown"
	}
	if cdr.CallID == "" {
		cdr.CallID = "unknown"
	}

	return cdr
}

func processAgentsData(rawBody []byte) ([]Agent, error) {
	var response AgentResponse
	if err := json.Unmarshal(rawBody, &response); err != nil {
		return nil, fmt.Errorf("failed to unmarshal agents response: %w", err)
	}

	if response.ErrCode != 0 {
		return nil, fmt.Errorf("agents fetch failed: %s", response.ErrMsg)
	}

	agents := make([]Agent, 0, len(response.Data))
	for _, rawAgent := range response.Data {
		cleanAgent := mapAgent(rawAgent)
		agents = append(agents, cleanAgent)
		log.Printf("Mapped agent: ID=%d, Name=%s, Number=%s, Presence=%s",
			cleanAgent.ID, cleanAgent.Name, cleanAgent.Number, cleanAgent.Presence)
	}

	return agents, nil
}

func processQueuesData(rawBody []byte) ([]Queue, error) {
	var response QueueResponse
	if err := json.Unmarshal(rawBody, &response); err != nil {
		return nil, fmt.Errorf("failed to unmarshal queues response: %w", err)
	}

	if response.ErrCode != 0 {
		return nil, fmt.Errorf("queues fetch failed: %s", response.ErrMsg)
	}

	queues := make([]Queue, 0, len(response.Data))
	for _, rawQueue := range response.Data {
		cleanQueue := mapQueue(rawQueue)
		queues = append(queues, cleanQueue)
		log.Printf("Mapped queue: Name=%s, Number=%s, RingStrategy=%s, SLATime=%d",
			cleanQueue.Name, cleanQueue.Number, cleanQueue.RingStrategy, cleanQueue.SLATime)
	}

	return queues, nil
}

func processCDRsData(rawBody []byte) ([]CDR, error) {
	var response CDRResponse
	if err := json.Unmarshal(rawBody, &response); err != nil {
		return nil, fmt.Errorf("failed to unmarshal CDRs response: %w", err)
	}

	if response.ErrCode != 0 {
		return nil, fmt.Errorf("CDRs fetch failed: %s", response.ErrMsg)
	}

	cdrs := make([]CDR, 0, len(response.Data))
	for _, rawCDR := range response.Data {
		cleanCDR := mapCDR(rawCDR)
		cdrs = append(cdrs, cleanCDR)
		log.Printf("Mapped CDR: ID=%d, From=%s, To=%s, Duration=%d, NoteType=%s, NoteDesc=%s",
			cleanCDR.ID, cleanCDR.CallFrom, cleanCDR.CallTo, cleanCDR.Duration,
			cleanCDR.CallNoteType, cleanCDR.CallNoteDescription)
	}

	return cdrs, nil
}
