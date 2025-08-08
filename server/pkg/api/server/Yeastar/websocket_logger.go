package Yeastar

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strings"
	"time"
)

const lastTimestampFile = "websocketLogger.txt"

func (cc *CortezaClient) OnSocketDisconnect() error {
	url := fmt.Sprintf("%s/api/gateway/socket/disconnect", cc.baseURL)

	fmt.Printf("[WebSocket_Logger] WebSocket Disconnected")
	fmt.Printf("[WebSocket_Logger] Sending log to: %s\n", url)

	resp, err := cc.client.Get(url)
	if err != nil {
		return fmt.Errorf("failed to create log: %w", err)
	}
	defer resp.Body.Close()

	if err := saveLastTimestamp(time.Now()); err != nil {
		log.Printf("Failed to save last timestamp on disconnect: %v", err)
	}

	return nil
}

type StartTime struct {
	CreatedAt string `json:"createdAt"`
}

func (cc *CortezaClient) OnSocketConnect() error {
	url := fmt.Sprintf("%s/api/gateway/socket/connect", cc.baseURL)

	fmt.Printf("[WebSocket_Logger] WebSocket Connected")
	fmt.Printf("[WebSocket_Logger] Sending log to: %s\n", url)

	httpResp, err := cc.client.Get(url)
	if err != nil {
		return fmt.Errorf("failed to create log: %w", err)
	}
	defer httpResp.Body.Close()

	body, _ := io.ReadAll(httpResp.Body)
	if httpResp.StatusCode != http.StatusOK {
		return fmt.Errorf("unexpected status %d: %s", httpResp.StatusCode, string(body))
	}

	fmt.Printf("[WebSocket_Logger] Response body: %s\n", string(body))

	//open calls - get the end ones
	cc.OpenCalls()

	// lost period
	var startResp StartTime
	err = json.Unmarshal([]byte(body), &startResp)
	if err != nil {
		log.Printf("failed to unmarshal response: %v", err)
	}

	layout := "2006-01-02 15:04:05 -0700 MST"
	remoteStartTime, err := time.Parse(layout, startResp.CreatedAt)
	if err != nil {
		log.Printf("Failed to parse createdAt: %v", err)
	}

	localStartTime, err := loadLastTimestamp()
	if err != nil {
		log.Printf("No local timestamp found or invalid, using remote start time")
		localStartTime = remoteStartTime
	}

	startTime := remoteStartTime
	if localStartTime.After(remoteStartTime) {
		startTime = localStartTime
	}

	fmt.Printf("[WebSocket_Logger] Final selected startTime: %s\n", startTime.Format("2006-01-02 15:04:05 -0700 MST"))

	endTime := time.Now()

	service, ctx, _, _ := setupSyncService(cc.baseURL)

	rawCDRsData, err := service.SearchMethodWithParameters(ctx, "cdr", startTime, endTime)
	if err != nil {
		return fmt.Errorf("failed to search cdrs: %w", err)
	}

	cdrs, err := processCDRsData(service, rawCDRsData)
	if err != nil {
		return fmt.Errorf("failed to process cdrs: %w", err)
	}

	if err := service.SendDataToCorteza(ctx, "cdr", cdrs); err != nil {
		return fmt.Errorf("failed to send cdrs to Corteza: %w", err)
	}

	if err := saveLastTimestamp(endTime); err != nil {
		log.Printf("Failed to save last timestamp: %v", err)
	}

	return nil
}

func saveLastTimestamp(t time.Time) error {
	return os.WriteFile(lastTimestampFile, []byte(t.Format(time.RFC3339)), 0644)
}

func loadLastTimestamp() (time.Time, error) {
	data, err := os.ReadFile(lastTimestampFile)
	if err != nil {
		return time.Time{}, err
	}
	return time.Parse(time.RFC3339, string(data))
}

func (cc *CortezaClient) OpenCalls() error {
	url := fmt.Sprintf("%s/api/gateway/socket/open-calls", cc.baseURL)

	httpResp, err := cc.client.Get(url)
	if err != nil {
		return fmt.Errorf("failed to create log: %w", err)
	}
	defer httpResp.Body.Close()

	body, _ := io.ReadAll(httpResp.Body)

	if httpResp.StatusCode != http.StatusOK {
		return fmt.Errorf("unexpected status %d: %s", httpResp.StatusCode, string(body))
	}

	if strings.HasPrefix(string(body), "&{") {
		fmt.Println("[WebSocket_Logger]: Empty or uninitialized data received")
		return nil
	}

	decoder := json.NewDecoder(bytes.NewReader(body))

	var oldestCall *CallResponse
	var oldestTime time.Time
	var callIDs []string

	for {
		var cr CallResponse
		err := decoder.Decode(&cr)
		if err != nil {
			if err == io.EOF {
				break // finished reading all objects
			}
			return fmt.Errorf("failed to decode JSON: %w", err)
		}
		callIDs = append(callIDs, cr.CallID)

		parsedTime, err := time.Parse("2006-01-02 15:04:05 -0700 MST", cr.CreatedAt)
		if err != nil {
			fmt.Printf("Failed to parse time for call ID %s: %v\n", cr.CallID, err)
			continue
		}

		if oldestCall == nil || parsedTime.Before(oldestTime) {
			oldestCall = &cr
			oldestTime = parsedTime
		}
	}

	fmt.Println("Collected call IDs:", callIDs)

	if oldestCall != nil {
		fmt.Printf("Oldest call is ID %s at %s\n", oldestCall.CallID, oldestCall.CreatedAt)
	} else {
		fmt.Println("No valid calls with parsable createdAt timestamps were found.")
	}

	fmt.Printf("[WebSocket_Logger] Final selected startTime: %s\n", oldestTime.Format("2006-01-02 15:04:05 -0700 MST"))

	endTime := time.Now()

	service, ctx, _, _ := setupSyncService(cc.baseURL)

	rawCDRsData, err := service.SearchMethodWithParameters(ctx, "cdr", oldestTime, endTime)
	if err != nil {
		return fmt.Errorf("failed to search cdrs: %w", err)
	}

	cdrs, err := processCDRsData(service, rawCDRsData)
	if err != nil {
		return fmt.Errorf("failed to process cdrs: %w", err)
	}

	callIDSet := make(map[string]bool, len(callIDs))
	for _, id := range callIDs {
		callIDSet[id] = true
	}

	var matchedCDRs []CDR
	for _, cdr := range cdrs {
		if callIDSet[cdr.CallID] {
			matchedCDRs = append(matchedCDRs, cdr)
		}
	}

	fmt.Printf("[WebSocket_Logger] Matched %d CDRs with open calls\n", len(matchedCDRs))

	if err := service.SendDataToCorteza(ctx, "cdr", matchedCDRs); err != nil {
		return fmt.Errorf("failed to send cdrs to Corteza: %w", err)
	}

	return nil
}

type CallResponse struct {
	CallID    string `json:"call_id"`
	CreatedAt string `json:"createdAt"`
}
