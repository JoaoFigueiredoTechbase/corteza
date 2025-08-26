package OpenAI

type TranscriptionRequestPayload struct {
	FileURL   string `json:"file_url"`
	OpenAIKey string `json:"openai_key"`
}

type SummaryRequestPayload struct {
	Transcription string `json:"transcription"`
	OpenAIKey     string `json:"openai_key"`
}

type WhisperResponse struct {
	Text string `json:"text"`
}

type RunResponse struct {
	ID       string `json:"id"`
	Status   string `json:"status"`
	ThreadID string `json:"thread_id"`
}

type MessageContent struct {
	Type string `json:"type"`
	Text struct {
		Value string `json:"value"`
	} `json:"text"`
}

type Message struct {
	ID          string           `json:"id"`
	Role        string           `json:"role"`
	Content     []MessageContent `json:"content"`
	AssistantID *string          `json:"assistant_id"`
}

type MessagesResponse struct {
	Data []Message `json:"data"`
}

type SummaryResponse struct {
	Assunto    string   `json:"Assunto"`
	Resumo     string   `json:"Resumo"`
	Topicos    []string `json:"Tópicos"`
	Satisfacao string   `json:"Satisfação"`
}
