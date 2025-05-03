package dalle

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"os"
)

// EnhancePrompt calls the OpenAI API to enhance a prompt using the given author type.
func EnhancePrompt(prompt, authorType string) (string, error) {
	apiKey := os.Getenv("OPENAI_API_KEY")
	return enhancePromptWithClient(prompt, authorType, &http.Client{}, apiKey, json.Marshal)
}

// enhancePromptWithClient is like EnhancePrompt but allows injecting an HTTP client, API key, and marshal function (for testing).
func enhancePromptWithClient(prompt, authorType string, client *http.Client, apiKey string, marshal func(v interface{}) ([]byte, error)) (string, error) {
	_ = authorType // authorType is not used in this function, but could be used for future enhancements
	url := "https://api.openai.com/v1/chat/completions"
	payload := dalleRequest{
		Model:     "gpt-4",
		Seed:      1337,
		Tempature: 0.2,
	}
	payload.Messages = append(payload.Messages, message{Role: "system", Content: prompt})
	payloadBytes, err := marshal(payload)
	if err != nil {
		return "", err
	}

	req, err := http.NewRequest("POST", url, bytes.NewBuffer(payloadBytes))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+apiKey)

	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	type dalleResponse struct {
		Choices []struct {
			Message message `json:"message"`
		} `json:"choices"`
	}
	var response dalleResponse
	if err := json.Unmarshal(body, &response); err != nil {
		return "", err
	} else {
		return response.Choices[0].Message.Content, nil
	}
}
