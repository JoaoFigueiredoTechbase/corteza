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
	"time"
)

type CDR struct {
	ID             int     `json:"id"`
	Time           string  `json:"time"`
	CallFrom       string  `json:"call_from"`
	CallTo         string  `json:"call_to"`
	Timestamp      int64   `json:"timestamp"`
	UID            string  `json:"uid"`
	SrcAddr        string  `json:"src_addr"`
	Duration       int     `json:"duration"`
	RingDuration   int     `json:"ring_duration"`
	TalkDuration   int     `json:"talk_duration"`
	Disposition    string  `json:"disposition"`
	CallType       string  `json:"call_type"`
	Reason         string  `json:"reason"`
	CallFromNumber string  `json:"call_from_number"`
	CallFromName   string  `json:"call_from_name"`
	CallToNumber   string  `json:"call_to_number"`
	CallToName     string  `json:"call_to_name"`
	CallID         string  `json:"call_id"`
	CallNote       *string `json:"call_note"`
	CallNoteID     string  `json:"call_note_id"`
	EnbCallNote    int     `json:"enb_call_note"`
	DID            string  `json:"did"`
	DIDName        string  `json:"did_name"`
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

func SyncCDR() ([]CDR, error) {
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
	var cdrs cdrResponse
	if err := json.Unmarshal(body, &cdrs); err != nil {
		return nil, err
	}
	if cdrs.ErrCode != 0 {
		return nil, fmt.Errorf("CDR fetch failed: %s", cdrs.ErrMsg)
	}

	// Step 3: Send to Corteza endpoint
	cortezaURL := "http://172.27.0.20:18080/api/gateway/cdr/update"

	payloadBytes, err := json.Marshal(cdrs)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal CDR payload: %w", err)
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

	return cdrs.Data, nil
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

	// cdrJSON, err := json.MarshalIndent(cdrs, "", "  ")
	// if err != nil {
	// 	log.Printf("Failed to marshal CDRs: %v\n", err)
	// } else {
	// 	log.Printf("CDR loaded:\n%s\n", string(cdrJSON))
	// }

}

func HandleSyncCDR(w http.ResponseWriter, r *http.Request) {
	cdrs, err := SyncCDR()
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to sync CDRs: %v", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(cdrs)
}
