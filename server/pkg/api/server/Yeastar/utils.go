package Yeastar

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path"
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

func mapQueue(raw QueueRaw) Queue {
	queue := Queue{
		ID:           raw.ID,
		Name:         safeStringConvert(raw.Name),
		Number:       safeStringConvert(raw.Number),
		RingStrategy: safeStringConvert(raw.RingStrategy),
		SLATime:      0, //safeIntConvert(raw.SLATime)
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

	queues := make([]Queue, 0, len(response.QueueList))
	for _, rawQueue := range response.QueueList {
		cleanQueue := mapQueue(rawQueue)
		queues = append(queues, cleanQueue)
		log.Printf("Mapped queue: Name=%s, Number=%s, RingStrategy=%s, SLATime=%d",
			cleanQueue.Name, cleanQueue.Number, cleanQueue.RingStrategy, cleanQueue.SLATime)
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
		log.Printf("Warning: failed to get recordings list: %v", err)
		// Continue without recordings if fetch fails
		recordings = []Recording{}
	}

	log.Printf("[DEBUG] Total recordings fetched: %d", len(recordings))
	log.Printf("[DEBUG] Total CDRs to process: %d", len(response.Data))

	// Create multiple mapping strategies for better matching
	recordingMaps := createRecordingMaps(recordings)

	cdrs := make([]CDR, 0, len(response.Data))
	matchedCount := 0

	for i, rawCDR := range response.Data {
		cleanCDR := mapCDR(rawCDR)

		log.Printf("[DEBUG] Processing CDR %d: ID=%d, UID=%s, RecordFile=%s",
			i+1, cleanCDR.ID, cleanCDR.UID, cleanCDR.RecordFile)

		// Try multiple matching strategies
		recording, matchType := findMatchingRecording(cleanCDR, recordingMaps)

		if recording != nil {
			downloadURL, err := service.GetRecordingDownloadURL(context.Background(), recording.ID)
			if err != nil {
				log.Printf("[WARN] Failed to get download URL for recording %d: %v", recording.ID, err)
			} else {
				cleanCDR.RecordingURL = downloadURL
				cleanCDR.RecordFile = recording.File
				matchedCount++
				log.Printf("[INFO] ✓ Matched CDR %d with recording %d via %s. URL: %s",
					cleanCDR.ID, recording.ID, matchType, downloadURL)
			}
		} else {
			log.Printf("[WARN] ✗ No matching recording found for CDR %d (UID: %s, RecordFile: %s)",
				cleanCDR.ID, cleanCDR.UID, cleanCDR.RecordFile)
		}

		cdrs = append(cdrs, cleanCDR)
	}

	log.Printf("[INFO] Successfully matched %d out of %d CDRs with recordings", matchedCount, len(cdrs))
	return cdrs, nil
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

// Create multiple mapping strategies for better matching
func createRecordingMaps(recordings []Recording) map[string]map[string]Recording {
	maps := make(map[string]map[string]Recording)

	// Map by UID (exact match)
	maps["uid"] = make(map[string]Recording)
	// Map by filename (basename)
	maps["filename"] = make(map[string]Recording)
	// Map by full file path
	maps["filepath"] = make(map[string]Recording)
	// Map by time-based key (if timestamps are similar)
	maps["time"] = make(map[string]Recording)

	for _, rec := range recordings {
		log.Printf("[DEBUG] Recording: ID=%d, UID=%s, File=%s, Time=%s",
			rec.ID, rec.UID, rec.File, rec.Time)

		// Map by UID
		if rec.UID != "" {
			maps["uid"][rec.UID] = rec
		}

		// Map by filename (basename)
		if rec.File != "" {
			basename := path.Base(rec.File)
			maps["filename"][basename] = rec
			// Also try without extension
			if ext := path.Ext(basename); ext != "" {
				nameWithoutExt := strings.TrimSuffix(basename, ext)
				maps["filename"][nameWithoutExt] = rec
			}
		}

		// Map by full file path
		if rec.File != "" {
			maps["filepath"][rec.File] = rec
		}

		// Map by time (you might need to adjust time format matching)
		if rec.Time != "" {
			maps["time"][rec.Time] = rec
		}
	}

	log.Printf("[DEBUG] Created mapping indices: uid=%d, filename=%d, filepath=%d, time=%d",
		len(maps["uid"]), len(maps["filename"]), len(maps["filepath"]), len(maps["time"]))

	return maps
}

// Try multiple strategies to find matching recording
func findMatchingRecording(cdr CDR, recordingMaps map[string]map[string]Recording) (*Recording, string) {
	// Strategy 1: Match by UID (most reliable)
	if cdr.UID != "" {
		if rec, exists := recordingMaps["uid"][cdr.UID]; exists {
			return &rec, "UID"
		}
	}

	// Strategy 2: Match by RecordFile basename
	if cdr.RecordFile != "" {
		basename := path.Base(cdr.RecordFile)
		if basename != "" && basename != "." {
			if rec, exists := recordingMaps["filename"][basename]; exists {
				return &rec, "filename"
			}

			// Try without extension
			if ext := path.Ext(basename); ext != "" {
				nameWithoutExt := strings.TrimSuffix(basename, ext)
				if rec, exists := recordingMaps["filename"][nameWithoutExt]; exists {
					return &rec, "filename_no_ext"
				}
			}
		}

		// Try full path
		if rec, exists := recordingMaps["filepath"][cdr.RecordFile]; exists {
			return &rec, "filepath"
		}
	}

	// Strategy 3: Match by time (if available and formatted similarly)
	if cdr.Time != "" {
		if rec, exists := recordingMaps["time"][cdr.Time]; exists {
			return &rec, "time"
		}
	}

	// Strategy 4: Fuzzy matching by call details (last resort)
	// This is more complex and might need adjustment based on your data
	for _, rec := range recordingMaps["uid"] {
		if fuzzyMatchCDRToRecording(cdr, rec) {
			return &rec, "fuzzy"
		}
	}

	return nil, ""
}

// Fuzzy matching based on call details
func fuzzyMatchCDRToRecording(cdr CDR, rec Recording) bool {
	// Match by call participants and approximate time
	// You might need to adjust this logic based on your data format

	// Check if call participants match
	callFromMatch := (cdr.CallFromNumber == rec.CallFromNumber) ||
		(cdr.CallFrom == rec.CallFrom)
	callToMatch := (cdr.CallToNumber == rec.CallToNumber) ||
		(cdr.CallTo == rec.CallTo)

	if !callFromMatch || !callToMatch {
		return false
	}

	// Check if durations are similar (within 5 seconds)
	durationDiff := abs(cdr.Duration - rec.Duration)
	if durationDiff > 5 {
		return false
	}

	// If we get here, it's likely a match
	log.Printf("[DEBUG] Fuzzy match found: CDR %d matches Recording %d", cdr.ID, rec.ID)
	return true
}

func abs(x int) int {
	if x < 0 {
		return -x
	}
	return x
}

// Enhanced debugging version of GetRecordingsList
func (ys *YeastarService) GetRecordingsListWithDebug(ctx context.Context) ([]Recording, error) {
	const endpoint = "recording"
	log.Printf("[INFO] Fetching recordings list from endpoint: %s", endpoint)

	rawData, err := ys.ListMethod(ctx, endpoint)
	if err != nil {
		log.Printf("[ERROR] Failed to fetch recordings: %v", err)
		return nil, fmt.Errorf("failed to fetch recordings: %w", err)
	}

	log.Printf("[DEBUG] Raw response data length: %d bytes", len(rawData))

	var response struct {
		ErrCode     int         `json:"errcode"`
		ErrMsg      string      `json:"errmsg"`
		TotalNumber int         `json:"total_number"`
		Data        []Recording `json:"data"`
	}

	if err := json.Unmarshal(rawData, &response); err != nil {
		log.Printf("[ERROR] Failed to unmarshal recordings: %v", err)
		log.Printf("[DEBUG] Raw data that failed to unmarshal: %s", string(rawData))
		return nil, fmt.Errorf("failed to unmarshal recordings: %w", err)
	}

	if response.ErrCode != 0 {
		log.Printf("[ERROR] Recordings fetch failed: %s", response.ErrMsg)
		return nil, fmt.Errorf("recordings fetch failed: %s", response.ErrMsg)
	}

	log.Printf("[INFO] Successfully fetched %d recordings (total reported: %d)",
		len(response.Data), response.TotalNumber)

	// Log first few recordings for debugging
	for i, rec := range response.Data {
		if i < 5 { // Log first 5 recordings
			log.Printf("[DEBUG] Recording %d: ID=%d, UID=%s, File=%s, CallFrom=%s, CallTo=%s, Duration=%d",
				i+1, rec.ID, rec.UID, rec.File, rec.CallFrom, rec.CallTo, rec.Duration)
		}
	}

	return response.Data, nil
}

// Diagnostic function to help debug the matching issue
func DiagnoseRecordingMatching(cdrs []CDR, recordings []Recording) {
	log.Printf("\n=== RECORDING MATCHING DIAGNOSIS ===")
	log.Printf("Total CDRs: %d", len(cdrs))
	log.Printf("Total Recordings: %d", len(recordings))

	// Analyze CDR patterns
	cdrWithRecordFile := 0
	cdrWithUID := 0
	for _, cdr := range cdrs {
		if cdr.RecordFile != "" {
			cdrWithRecordFile++
		}
		if cdr.UID != "" {
			cdrWithUID++
		}
	}

	log.Printf("CDRs with RecordFile: %d", cdrWithRecordFile)
	log.Printf("CDRs with UID: %d", cdrWithUID)

	// Analyze Recording patterns
	recWithFile := 0
	recWithUID := 0
	for _, rec := range recordings {
		if rec.File != "" {
			recWithFile++
		}
		if rec.UID != "" {
			recWithUID++
		}
	}

	log.Printf("Recordings with File: %d", recWithFile)
	log.Printf("Recordings with UID: %d", recWithUID)

	// Sample data comparison
	if len(cdrs) > 0 && len(recordings) > 0 {
		log.Printf("\nSample CDR: ID=%d, UID='%s', RecordFile='%s'",
			cdrs[0].ID, cdrs[0].UID, cdrs[0].RecordFile)
		log.Printf("Sample Recording: ID=%d, UID='%s', File='%s'",
			recordings[0].ID, recordings[0].UID, recordings[0].File)
	}
	log.Printf("=== END DIAGNOSIS ===\n")
}
