package OpenAI

import (
	"bytes"
	"encoding/json"
	"io"
	"log"
	"mime/multipart"
	"net/http"
)

func HandleTranscription(w http.ResponseWriter, r *http.Request) {
	log.Println("Received transcription request")

	var payload TranscriptionRequestPayload
	err := json.NewDecoder(r.Body).Decode(&payload)
	if err != nil {
		log.Printf("ERROR: Failed to decode request body: %v\n", err)
		http.Error(w, "invalid request", http.StatusBadRequest)
		return
	}
	log.Printf("Request payload: %+v\n", payload)

	resp, err := http.Get(payload.FileURL)
	if err != nil {
		log.Printf("ERROR: Failed to download file from %s: %v\n", payload.FileURL, err)
		http.Error(w, "failed to download file", http.StatusInternalServerError)
		return
	}
	defer resp.Body.Close()
	log.Println("File downloaded successfully")

	buf := new(bytes.Buffer)
	_, err = io.Copy(buf, resp.Body)
	if err != nil {
		log.Printf("ERROR: Failed to read downloaded file: %v\n", err)
		http.Error(w, "failed to read file", http.StatusInternalServerError)
		return
	}
	log.Printf("File size: %d bytes\n", buf.Len())

	var b bytes.Buffer
	writer := multipart.NewWriter(&b)
	part, err := writer.CreateFormFile("file", "audio.wav")
	if err != nil {
		log.Printf("ERROR: Failed to create form file: %v\n", err)
		http.Error(w, "failed to create form file", http.StatusInternalServerError)
		return
	}

	_, err = io.Copy(part, bytes.NewReader(buf.Bytes()))
	if err != nil {
		log.Printf("ERROR: Failed to write file to multipart form: %v\n", err)
		http.Error(w, "failed to write file to form", http.StatusInternalServerError)
		return
	}
	log.Println("File added to multipart form successfully")

	writer.WriteField("model", "whisper-1")
	writer.Close()

	req, _ := http.NewRequest("POST", "https://api.openai.com/v1/audio/transcriptions", &b)
	req.Header.Set("Authorization", "Bearer "+payload.OpenAIKey)
	req.Header.Set("Content-Type", writer.FormDataContentType())

	log.Println("Sending request to OpenAI Whisper API")
	client := &http.Client{}
	resp2, err := client.Do(req)
	if err != nil {
		log.Printf("ERROR: Failed to call OpenAI API: %v\n", err)
		http.Error(w, "failed to call OpenAI API", http.StatusInternalServerError)
		return
	}
	defer resp2.Body.Close()
	log.Printf("Received response from OpenAI API with status: %s\n", resp2.Status)

	if resp2.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp2.Body)
		log.Printf("OpenAI API error (%d): %s", resp2.StatusCode, string(bodyBytes))
		http.Error(w, "OpenAI API error: "+string(bodyBytes), resp2.StatusCode)
		return
	}

	var whisperResp WhisperResponse
	err = json.NewDecoder(resp2.Body).Decode(&whisperResp)
	if err != nil {
		log.Printf("ERROR: Failed to decode OpenAI response: %v\n", err)
		http.Error(w, "failed to decode response", http.StatusInternalServerError)
		return
	}
	log.Printf("Transcription received: %+v\n", whisperResp)

	w.Header().Set("Content-Type", "application/json")
	err = json.NewEncoder(w).Encode(whisperResp)
	if err != nil {
		log.Printf("ERROR: Failed to send response: %v\n", err)
	} else {
		log.Println("Transcription response sent successfully")
	}
}

func HandleCallSummary(w http.ResponseWriter, r *http.Request) {
	log.Println("Received call summary request")

	assistantID := "asst_Vmkjrm11ZQ7xirDIiJaHNrsZ"

	var payload SummaryRequestPayload
	err := json.NewDecoder(r.Body).Decode(&payload)
	if err != nil {
		log.Printf("ERROR: Failed to decode request body: %v\n", err)
		http.Error(w, "invalid request", http.StatusBadRequest)
		return
	}

	threadID, runID, err := createThreadAndRun(assistantID, payload.Transcription, payload.OpenAIKey)
	if err != nil {
		log.Printf("ERROR: Failed to create thread and run: %v\n", err)
		http.Error(w, "failed to create thread and run", http.StatusInternalServerError)
		return
	}

	log.Printf("Created thread: %s, run: %s", threadID, runID)

	err = waitForRunCompletion(threadID, runID, payload.OpenAIKey)
	if err != nil {
		log.Printf("ERROR: Run failed: %v\n", err)
		http.Error(w, "run execution failed", http.StatusInternalServerError)
		return
	}

	log.Println("Run completed successfully")

	summaryResponse, err := getMessagesAndExtractSummary(threadID, payload.OpenAIKey)
	if err != nil {
		log.Printf("ERROR: Failed to get summary: %v\n", err)
		http.Error(w, "failed to get summary", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(summaryResponse)
	log.Println("Summary response sent successfully")
}
