package Yeastar

import (
	"bytes"
	"crypto/tls"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"
)

type Agent struct {
	ID       int    `json:"id"`
	Presence string `json:"presence_status"`
	Number   string `json:"number"`
	Name     string `json:"caller_id_name"`
}

type Queue struct {
	Name         string `json:"name"`
	Number       string `json:"number"`
	RingStrategy string `json:"ring_strategy"`
	SLATime      int    `json:"sla_time"`
}

type CDR struct {
	ID             int    `json:"id"`
	Time           string `json:"time"`
	CallFrom       string `json:"call_from"`
	CallTo         string `json:"call_to"`
	Timestamp      int64  `json:"timestamp"`
	UID            string `json:"uid"`
	SrcAddr        string `json:"src_addr"`
	Duration       int    `json:"duration"`
	RingDuration   int    `json:"ring_duration"`
	TalkDuration   int    `json:"talk_duration"`
	Disposition    string `json:"disposition"`
	CallType       string `json:"call_type"`
	Reason         string `json:"reason"`
	CallFromNumber string `json:"call_from_number"`
	CallFromName   string `json:"call_from_name"`
	CallToNumber   string `json:"call_to_number"`
	CallToName     string `json:"call_to_name"`
	CallID         string `json:"call_id"`
	CallNote       string `json:"call_note"`
	CallNoteID     string `json:"call_note_id"`
	EnbCallNote    int    `json:"enb_call_note"`
	DID            string `json:"did"`
	DIDName        string `json:"did_name"`
}

type agentResponse struct {
	ErrCode     int        `json:"errcode"`
	ErrMsg      string     `json:"errmsg"`
	TotalNumber int        `json:"total_number"`
	Data        []rawAgent `json:"data"`
}

type rawAgent struct {
	ID                   int                    `json:"id"`
	OnlineStatus         map[string]interface{} `json:"online_status"`
	PresenceStatus       string                 `json:"presence_status"`
	Number               string                 `json:"number"`
	CallerIDName         string                 `json:"caller_id_name"`
	Timezone             string                 `json:"timezone"`
	CustomPresenceStatus string                 `json:"custom_presence_status"`
}

type queueResponse struct {
	ErrCode     int        `json:"errcode"`
	ErrMsg      string     `json:"errmsg"`
	TotalNumber int        `json:"total_number"`
	Data        []rawQueue `json:"data"`
}

type rawQueue struct {
	Name         interface{} `json:"name"`
	Number       interface{} `json:"number"`
	RingStrategy interface{} `json:"ring_strategy"`
	SLATime      interface{} `json:"sla_time"`
}

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

func mapAgent(raw rawAgent) Agent {
	agent := Agent{
		ID:       raw.ID,
		Presence: safeStringConvert(raw.PresenceStatus),
		Number:   safeStringConvert(raw.Number),
		Name:     safeStringConvert(raw.CallerIDName),
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

	return agent
}

// mapQueue cleans and maps raw queue data to the clean Queue structure
func mapQueue(raw rawQueue) Queue {
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

// mapCDR cleans and maps raw CDR data to the clean CleanedCDR structure
func mapCDR(raw CDR) CDR {
	cdr := CDR{
		ID:             raw.ID,
		Time:           safeStringConvert(raw.Time),
		CallFrom:       safeStringConvert(raw.CallFrom),
		CallTo:         safeStringConvert(raw.CallTo),
		Timestamp:      safeInt64Convert(raw.Timestamp),
		UID:            safeStringConvert(raw.UID),
		SrcAddr:        safeStringConvert(raw.SrcAddr),
		Duration:       safeIntConvert(raw.Duration),
		RingDuration:   safeIntConvert(raw.RingDuration),
		TalkDuration:   safeIntConvert(raw.TalkDuration),
		Disposition:    safeStringConvert(raw.Disposition),
		CallType:       safeStringConvert(raw.CallType),
		Reason:         safeStringConvert(raw.Reason),
		CallFromNumber: safeStringConvert(raw.CallFromNumber),
		CallFromName:   safeStringConvert(raw.CallFromName),
		CallToNumber:   safeStringConvert(raw.CallToNumber),
		CallToName:     safeStringConvert(raw.CallToName),
		CallID:         safeStringConvert(raw.CallID),
		CallNote:       safeStringConvert(raw.CallNote),
		CallNoteID:     safeStringConvert(raw.CallNoteID),
		EnbCallNote:    safeIntConvert(raw.EnbCallNote),
		DID:            safeStringConvert(raw.DID),
		DIDName:        safeStringConvert(raw.DIDName),
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

// processAgentsData processes and cleans agent data from the API response
func processAgentsData(rawBody []byte) ([]Agent, error) {
	var response agentResponse
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

// processQueuesData processes and cleans queue data from the API response
func processQueuesData(rawBody []byte) ([]Queue, error) {
	var response queueResponse
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

// processCDRsData processes and cleans CDR data from the API response
func processCDRsData(rawBody []byte) ([]CDR, error) {
	var response cdrResponse
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
		log.Printf("Mapped CDR: ID=%d, From=%s, To=%s, Duration=%d",
			cleanCDR.ID, cleanCDR.CallFrom, cleanCDR.CallTo, cleanCDR.Duration)
	}

	return cdrs, nil
}

type tokenResponse struct {
	ErrCode     int    `json:"errcode"`
	ErrMsg      string `json:"errmsg"`
	AccessToken string `json:"access_token"`
}

type cdrResponse struct {
	ErrCode     int    `json:"errcode"`
	ErrMsg      string `json:"errmsg"`
	TotalNumber int    `json:"total_number"`
	Data        []CDR  `json:"data"`
}

const (
	apiURL      = "https://172.26.0.6:8088/"
	apiUsername = "eOoVHNLBl0ytb6sM19HVHVDKKwDNoxsS"
	apiPassword = "YyclbdWjDcmNBPvviNMG2eeuB3oZAqnj"
)

func FetchCDRs() ([]CDR, error) {
	// Get token
	tokenPayload := map[string]string{
		"username": apiUsername,
		"password": apiPassword,
	}
	tokenBody, _ := json.Marshal(tokenPayload)

	req, err := http.NewRequest("POST", fmt.Sprintf("%s/openapi/v1.0/get_token", apiURL), bytes.NewReader(tokenBody))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", "OpenAPI")

	client := &http.Client{
		Timeout: 10 * time.Second,
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
		},
	}
	res, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()

	var tokenRes tokenResponse
	if err := json.NewDecoder(res.Body).Decode(&tokenRes); err != nil {
		return nil, err
	}
	if tokenRes.ErrCode != 0 {
		return nil, errors.New("failed to retrieve token")
	}

	// Fetch CDRs
	cdrURL := fmt.Sprintf("%s/openapi/v1.0/cdr/list?access_token=%s", apiURL, tokenRes.AccessToken)
	req2, err := http.NewRequest("GET", cdrURL, nil)
	if err != nil {
		return nil, err
	}
	req2.Header.Set("Accept", "application/json")
	req2.Header.Set("User-Agent", "OpenAPI")

	res2, err := client.Do(req2)
	if err != nil {
		return nil, err
	}
	defer res2.Body.Close()

	body, _ := io.ReadAll(res2.Body)
	var cdrs cdrResponse
	if err := json.Unmarshal(body, &cdrs); err != nil {
		return nil, err
	}
	if cdrs.ErrCode != 0 {
		return nil, fmt.Errorf("CDR fetch failed: %s", cdrs.ErrMsg)
	}

	db, err := sql.Open("postgres", "postgres://postgres:12345678@localhost:5432/corteza?sslmode=disable")
	if err != nil {
		return nil, fmt.Errorf("db open error: %w", err)
	}
	defer db.Close()

	insertStmt := `
	INSERT INTO cdrs (
		time, call_from, call_to, timestamp, uid, src_addr, duration,
		talk_duration, disposition, call_type, reason,
		call_from_number, call_from_name, call_to_number, call_to_name,
		call_id, call_note, call_note_id, enb_call_note, did, did_name
	) VALUES (
		$1, $2, $3, $4, $5, $6, $7,
		$8, $9, $10, $11,
		$12, $13, $14, $15,
		$16, $17, $18, $19, $20, $21
	)`

	for _, cdr := range cdrs.Data {
		_, err := db.Exec(insertStmt,
			cdr.Time, cdr.CallFrom, cdr.CallTo, cdr.Timestamp, cdr.UID,
			cdr.SrcAddr, cdr.Duration, cdr.TalkDuration, cdr.Disposition,
			cdr.CallType, cdr.Reason, cdr.CallFromNumber, cdr.CallFromName,
			cdr.CallToNumber, cdr.CallToName, cdr.CallID,
			cdr.CallNote, cdr.CallNoteID, cdr.EnbCallNote, cdr.DID, cdr.DIDName,
		)
		if err != nil {
			log.Printf("Failed to insert CDR %s: %v\n", cdr.UID, err)
		}
	}

	fmt.Println("CDRs successfully inserted into database")

	return cdrs.Data, nil
}

func LoadCDRsFromDB() ([]CDR, error) {
	db, err := sql.Open("postgres", "postgres://postgres:12345678@localhost:5432/corteza?sslmode=disable")
	if err != nil {
		return nil, fmt.Errorf("db open error: %w", err)
	}
	defer db.Close()

	rows, err := db.Query(`
		SELECT 
			id, time, call_from, call_to, timestamp, uid, src_addr, duration,
			talk_duration, disposition, call_type, reason,
			call_from_number, call_from_name, call_to_number, call_to_name,
			call_id, call_note, call_note_id, enb_call_note, did, did_name
		FROM cdrs
	`)
	if err != nil {
		return nil, fmt.Errorf("query error: %w", err)
	}
	defer rows.Close()

	var cdrs []CDR
	for rows.Next() {
		var cdr CDR
		err := rows.Scan(
			&cdr.ID, &cdr.Time, &cdr.CallFrom, &cdr.CallTo, &cdr.Timestamp, &cdr.UID,
			&cdr.SrcAddr, &cdr.Duration, &cdr.TalkDuration, &cdr.Disposition,
			&cdr.CallType, &cdr.Reason, &cdr.CallFromNumber, &cdr.CallFromName,
			&cdr.CallToNumber, &cdr.CallToName, &cdr.CallID, &cdr.CallNote,
			&cdr.CallNoteID, &cdr.EnbCallNote, &cdr.DID, &cdr.DIDName,
		)
		if err != nil {
			log.Printf("row scan error: %v", err)
			continue
		}
		cdrs = append(cdrs, cdr)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("rows iteration error: %w", err)
	}

	return cdrs, nil
}
func SyncAgentWithMapper() ([]Agent, error) {
	// Step 1: Get access token
	tokenPayload := map[string]string{
		"username": apiUsername,
		"password": apiPassword,
	}
	tokenBody, _ := json.Marshal(tokenPayload)

	req, err := http.NewRequest("POST", fmt.Sprintf("%s/openapi/v1.0/get_token", apiURL), bytes.NewReader(tokenBody))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", "OpenAPI")

	client := &http.Client{
		Timeout: 10 * time.Second,
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
		},
	}

	res, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()

	var tokenRes tokenResponse
	if err := json.NewDecoder(res.Body).Decode(&tokenRes); err != nil {
		return nil, err
	}
	if tokenRes.ErrCode != 0 {
		return nil, errors.New("failed to retrieve token")
	}

	// Step 2: Fetch extensions
	extensionsURL := fmt.Sprintf("%s/openapi/v1.0/extension/list?access_token=%s", apiURL, tokenRes.AccessToken)
	req2, err := http.NewRequest("GET", extensionsURL, nil)
	if err != nil {
		return nil, err
	}
	req2.Header.Set("Accept", "application/json")
	req2.Header.Set("User-Agent", "OpenAPI")

	res2, err := client.Do(req2)
	if err != nil {
		return nil, err
	}
	defer res2.Body.Close()

	body, err := io.ReadAll(res2.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read agents response: %w", err)
	}

	// Step 3: Process and clean the data
	agents, err := processAgentsData(body)
	if err != nil {
		return nil, err
	}

	// Step 4: Send to Corteza endpoint
	cortezaURL := "http://localhost:80/api/gateway/agent/sync"

	payloadBytes, err := json.Marshal(agents)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal agents payload: %w", err)
	}

	req3, err := http.NewRequest("POST", cortezaURL, bytes.NewBuffer(payloadBytes))
	if err != nil {
		return nil, fmt.Errorf("failed to create Corteza request: %w", err)
	}
	req3.Header.Set("Content-Type", "application/json")

	res3, err := client.Do(req3)
	if err != nil {
		return nil, fmt.Errorf("failed to send agents to Corteza: %w", err)
	}
	defer res3.Body.Close()

	if res3.StatusCode >= 300 {
		respBody, _ := io.ReadAll(res3.Body)
		return nil, fmt.Errorf("Corteza rejected the request: %s", string(respBody))
	}

	return agents, nil
}

// SyncQueueWithMapper syncs queues with data cleaning and mapping
func SyncQueueWithMapper() ([]Queue, error) {
	// Step 1: Get access token
	tokenPayload := map[string]string{
		"username": apiUsername,
		"password": apiPassword,
	}
	tokenBody, _ := json.Marshal(tokenPayload)

	req, err := http.NewRequest("POST", fmt.Sprintf("%s/openapi/v1.0/get_token", apiURL), bytes.NewReader(tokenBody))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", "OpenAPI")

	client := &http.Client{
		Timeout: 10 * time.Second,
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
		},
	}

	res, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()

	var tokenRes tokenResponse
	if err := json.NewDecoder(res.Body).Decode(&tokenRes); err != nil {
		return nil, err
	}
	if tokenRes.ErrCode != 0 {
		return nil, errors.New("failed to retrieve token")
	}

	// Step 2: Fetch queues
	queueURL := fmt.Sprintf("%s/openapi/v1.0/queue/list?access_token=%s", apiURL, tokenRes.AccessToken)
	req2, err := http.NewRequest("GET", queueURL, nil)
	if err != nil {
		return nil, err
	}
	req2.Header.Set("Accept", "application/json")
	req2.Header.Set("User-Agent", "OpenAPI")

	res2, err := client.Do(req2)
	if err != nil {
		return nil, err
	}
	defer res2.Body.Close()

	body, err := io.ReadAll(res2.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read queues response: %w", err)
	}

	// Step 3: Process and clean the data
	queues, err := processQueuesData(body)
	if err != nil {
		return nil, err
	}

	// Step 4: Send to Corteza endpoint
	cortezaURL := "http://localhost:80/api/gateway/queue/sync"

	payloadBytes, err := json.Marshal(queues)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal queues payload: %w", err)
	}

	req3, err := http.NewRequest("POST", cortezaURL, bytes.NewBuffer(payloadBytes))
	if err != nil {
		return nil, fmt.Errorf("failed to create Corteza request: %w", err)
	}
	req3.Header.Set("Content-Type", "application/json")

	res3, err := client.Do(req3)
	if err != nil {
		return nil, fmt.Errorf("failed to send queues to Corteza: %w", err)
	}
	defer res3.Body.Close()

	if res3.StatusCode >= 300 {
		respBody, _ := io.ReadAll(res3.Body)
		return nil, fmt.Errorf("Corteza rejected the request: %s", string(respBody))
	}

	return queues, nil
}

// SyncCDRWithMapper syncs CDRs with data cleaning and mapping
func SyncCDRWithMapper() ([]CDR, error) {
	// Step 1: Get access token
	tokenPayload := map[string]string{
		"username": apiUsername,
		"password": apiPassword,
	}
	tokenBody, _ := json.Marshal(tokenPayload)

	req, err := http.NewRequest("POST", fmt.Sprintf("%s/openapi/v1.0/get_token", apiURL), bytes.NewReader(tokenBody))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", "OpenAPI")

	client := &http.Client{
		Timeout: 10 * time.Second,
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
		},
	}

	res, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()

	var tokenRes tokenResponse
	if err := json.NewDecoder(res.Body).Decode(&tokenRes); err != nil {
		return nil, err
	}
	if tokenRes.ErrCode != 0 {
		return nil, errors.New("failed to retrieve token")
	}

	// Step 2: Fetch CDRs
	cdrURL := fmt.Sprintf("%s/openapi/v1.0/cdr/list?access_token=%s", apiURL, tokenRes.AccessToken)
	req2, err := http.NewRequest("GET", cdrURL, nil)
	if err != nil {
		return nil, err
	}
	req2.Header.Set("Accept", "application/json")
	req2.Header.Set("User-Agent", "OpenAPI")

	res2, err := client.Do(req2)
	if err != nil {
		return nil, err
	}
	defer res2.Body.Close()

	body, err := io.ReadAll(res2.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read CDR response: %w", err)
	}

	// Step 3: Process and clean the data
	cdrs, err := processCDRsData(body)
	if err != nil {
		return nil, err
	}

	// Step 4: Send to Corteza endpoint
	cortezaURL := "http://localhost:80/api/gateway/cdr/sync"

	payloadBytes, err := json.Marshal(cdrs)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal CDRs payload: %w", err)
	}

	req3, err := http.NewRequest("POST", cortezaURL, bytes.NewBuffer(payloadBytes))
	if err != nil {
		return nil, fmt.Errorf("failed to create Corteza request: %w", err)
	}
	req3.Header.Set("Content-Type", "application/json")

	res3, err := client.Do(req3)
	if err != nil {
		return nil, fmt.Errorf("failed to send CDRs to Corteza: %w", err)
	}
	defer res3.Body.Close()

	if res3.StatusCode >= 300 {
		respBody, _ := io.ReadAll(res3.Body)
		return nil, fmt.Errorf("Corteza rejected the request: %s", string(respBody))
	}

	return cdrs, nil
}

func HandleFetchCDRs(w http.ResponseWriter, r *http.Request) {
	cdrs, err := FetchCDRs()
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to fetch CDRs: %v", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(cdrs)
}

func HandleCDRDB(w http.ResponseWriter, r *http.Request) {
	cdrs, err := LoadCDRsFromDB()
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to load CDRs from DB: %v", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(cdrs)
}

func HandleSyncCDR(w http.ResponseWriter, r *http.Request) {
	cdrs, err := SyncCDRWithMapper()
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to sync CDRs: %v", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(cdrs)
}

func HandleSyncAgent(w http.ResponseWriter, r *http.Request) {
	cdrs, err := SyncAgentWithMapper()
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to sync Agent: %v", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(cdrs)
}

func HandleSyncQueue(w http.ResponseWriter, r *http.Request) {
	cdrs, err := SyncQueueWithMapper()
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to sync Queues: %v", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(cdrs)
}
