package Yeastar

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"
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
		return v, "", ""
	case map[string]interface{}:
		if remark, ok := v["remark"].(string); ok {
			noteDescription = strings.TrimSpace(remark)
		}

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
		ID:             raw.ID,
		Presence:       safeStringConvert(raw.Presence),
		Number:         safeStringConvert(raw.Number),
		Name:           safeStringConvert(raw.Name),
		Email:          safeStringConvert(raw.Email),
		CustomPresence: safeStringConvert(raw.CustomPresence),
	}

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
		agent.Email = ""
	}
	if agent.CustomPresence == "" {
		agent.CustomPresence = ""
	}

	return agent
}

func mapQueue(raw QueueRaw) Queue {
	queue := Queue{
		ID:           raw.ID,
		Name:         safeStringConvert(raw.Name),
		Number:       safeStringConvert(raw.Number),
		RingStrategy: safeStringConvert(raw.RingStrategy),
		SLATime:      0, //safeIntConvert(raw.SLATime)
	}

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
		queue.SLATime = 60
	}

	return queue
}

func mapCDR(raw CDR) CDR {
	noteText, noteType, noteDescription := extractCallNoteInfo(raw.CallNote)

	unixTimeStamp := safeInt64Convert(raw.Timestamp)
	timeStampIso := time.Unix(unixTimeStamp, 0).UTC()

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
		TimeCorteza:         timeStampIso,
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
		//log.Printf("Mapped agent: ID=%d, Name=%s, Number=%s, Presence=%s",cleanAgent.ID, cleanAgent.Name, cleanAgent.Number, cleanAgent.Presence)
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

	queues := make([]Queue, 0, len(response.QueueList))
	for _, rawQueue := range response.QueueList {
		cleanQueue := mapQueue(rawQueue)
		queues = append(queues, cleanQueue)
		//log.Printf("Mapped queue: Name=%s, Number=%s, RingStrategy=%s, SLATime=%d",cleanQueue.Name, cleanQueue.Number, cleanQueue.RingStrategy, cleanQueue.SLATime)
	}

	return queues, nil
}

func processCDRsData(service *YeastarService, rawBody []byte) ([]CDR, error) {
	var response CDRResponse
	if err := json.Unmarshal(rawBody, &response); err != nil {
		return nil, fmt.Errorf("failed to unmarshal CDRs response: %w", err)
	}

	if response.ErrCode != 0 {
		return nil, fmt.Errorf("CDRs fetch failed: %s", response.ErrMsg)
	}

	// Get recordings list
	recordings, err := service.GetRecordingsList(context.Background())
	if err != nil {
		//log.Printf("Warning: failed to get recordings list: %v", err)
		// Continue without recordings if fetch fails
		recordings = []Recording{}
	}

	cdrs := make([]CDR, 0, len(response.Data))
	for _, rawCDR := range response.Data {
		cleanCDR := mapCDR(rawCDR)
		cdrs = append(cdrs, cleanCDR)
	}

	updatedCDRs := MergeRecordingsWithCDRs(recordings, cdrs)

	return updatedCDRs, nil
}

func processQueueMembersData(rawBody []byte) ([]QueueMember, error) {
	var response struct {
		ErrCode   int        `json:"errcode"`
		ErrMsg    string     `json:"errmsg"`
		QueueList []QueueRaw `json:"queue_list"`
	}

	if err := json.Unmarshal(rawBody, &response); err != nil {
		return nil, fmt.Errorf("failed to unmarshal queue members: %w", err)
	}
	if response.ErrCode != 0 {
		return nil, fmt.Errorf("queue members fetch failed: %s", response.ErrMsg)
	}

	var allMembers []QueueMember
	for _, queue := range response.QueueList {
		members := mapQueueMembers(queue)
		allMembers = append(allMembers, members...)
	}

	// err := writeQueueMembersToFile("queue_members.json", allMembers)
	// if err != nil {
	// 	log.Fatalf("Error writing to file: %v", err)
	// }

	return allMembers, nil
}

func mapQueueMembers(queue QueueRaw) []QueueMember {
	var members []QueueMember

	extractMembers := func(agentList []AgentEntry, memberType string) {
		for _, entry := range agentList {
			agentID := safeIntConvert(entry.Value)
			members = append(members, QueueMember{
				QueueID:     queue.ID,
				QueueName:   queue.Name,
				QueueNumber: queue.Number,
				AgentID:     agentID,
				AgentExt:    entry.Text,
				Type:        memberType,
			})
		}
	}

	extractMembers(queue.DynamicAgents, "dynamic")
	extractMembers(queue.StaticAgents, "static")
	extractMembers(queue.Managers, "manager")

	return members
}

func dumpCDRsToFile(cdrs []CDR) error {
	filename := fmt.Sprintf("cdrs_dump_%s.json", time.Now().Format("20060102_150405"))
	file, err := os.Create(filename)
	if err != nil {
		return fmt.Errorf("failed to create dump file: %w", err)
	}
	defer file.Close()

	encoder := json.NewEncoder(file)
	encoder.SetIndent("", "  ")

	if err := encoder.Encode(cdrs); err != nil {
		return fmt.Errorf("failed to write CDRs to file: %w", err)
	}

	fmt.Println("CDRs written to", filename)
	return nil
}

func MergeRecordingsWithCDRs(recordings []Recording, cdrs []CDR) []CDR {
	recMap := make(map[int]string)
	for _, rec := range recordings {
		recMap[rec.ID] = rec.File
	}

	for i, cdr := range cdrs {
		if file, ok := recMap[cdr.ID]; ok {
			cdrs[i].RecordFile = file
		}
	}

	return cdrs
}
