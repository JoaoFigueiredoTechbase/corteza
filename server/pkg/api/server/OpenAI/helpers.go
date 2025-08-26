package OpenAI

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"regexp"
	"strings"
	"time"
)

func waitForRunCompletion(threadID, runID, apiKey string) error {
	url := fmt.Sprintf("https://api.openai.com/v1/threads/%s/runs/%s", threadID, runID)

	client := &http.Client{Timeout: 30 * time.Second}

	for i := 0; i < 30; i++ {
		req, _ := http.NewRequest("GET", url, nil)
		req.Header.Set("Authorization", "Bearer "+apiKey)

		resp, err := client.Do(req)
		if err != nil {
			return err
		}

		var runResp RunResponse
		err = json.NewDecoder(resp.Body).Decode(&runResp)
		resp.Body.Close()

		if err != nil {
			return err
		}

		log.Printf("Run status: %s", runResp.Status)

		switch runResp.Status {
		case "completed":
			return nil
		case "failed", "cancelled", "expired":
			return fmt.Errorf("run failed with status: %s", runResp.Status)
		case "queued", "in_progress":
			time.Sleep(10 * time.Second)
			continue
		default:
			time.Sleep(5 * time.Second)
		}
	}

	return fmt.Errorf("run did not complete within timeout")
}

func getMessagesAndExtractSummary(threadID, apiKey string) (*SummaryResponse, error) {
	url := fmt.Sprintf("https://api.openai.com/v1/threads/%s/messages", threadID)

	req, _ := http.NewRequest("GET", url, nil)
	req.Header.Set("Authorization", "Bearer "+apiKey)

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("API error: %s", string(bodyBytes))
	}

	var messagesResp MessagesResponse
	err = json.NewDecoder(resp.Body).Decode(&messagesResp)
	if err != nil {
		return nil, err
	}

	for _, message := range messagesResp.Data {
		if message.Role == "assistant" && message.AssistantID != nil {
			if len(message.Content) > 0 {
				return extractJSONFromText(message.Content[0].Text.Value)
			}
		}
	}

	return nil, fmt.Errorf("no assistant response found")
}

func extractJSONFromText(text string) (*SummaryResponse, error) {
	re := regexp.MustCompile("```json\\s*([\\s\\S]*?)\\s*```")
	matches := re.FindStringSubmatch(text)

	var jsonStr string
	if len(matches) > 1 {
		jsonStr = matches[1]
	} else {
		jsonStr = strings.TrimSpace(text)
	}

	var summary SummaryResponse
	err := json.Unmarshal([]byte(jsonStr), &summary)
	if err != nil {
		return nil, fmt.Errorf("failed to parse JSON: %v", err)
	}

	return &summary, nil
}

func createThreadAndRun(assistantID, transcription, apiKey string) (string, string, error) {
	reqBody := map[string]interface{}{
		"assistant_id": assistantID,
		"thread": map[string]interface{}{
			"messages": []map[string]string{
				{
					"role":    "user",
					"content": transcription,
				},
			},
		},
	}

	jsonBody, _ := json.Marshal(reqBody)

	req, _ := http.NewRequest("POST", "https://api.openai.com/v1/threads/runs", bytes.NewBuffer(jsonBody))
	req.Header.Set("Authorization", "Bearer "+apiKey)
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 60 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return "", "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return "", "", fmt.Errorf("API error: %s", string(bodyBytes))
	}

	var runResp RunResponse
	err = json.NewDecoder(resp.Body).Decode(&runResp)
	if err != nil {
		return "", "", err
	}

	return runResp.ThreadID, runResp.ID, nil
}
