package dalle

import "encoding/json"

type message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type dalleRequest struct {
	Input     string    `json:"input,omitempty"`
	Prompt    string    `json:"prompt,omitempty"`
	N         int       `json:"n,omitempty"`
	Quality   string    `json:"quality,omitempty"`
	Model     string    `json:"model,omitempty"`
	Style     string    `json:"style,omitempty"`
	Size      string    `json:"size,omitempty"`
	Seed      int       `json:"seed,omitempty"`
	Tempature float64   `json:"temperature,omitempty"`
	Messages  []message `json:"messages,omitempty"`
}

func (req *dalleRequest) String() string {
	bytes, _ := json.MarshalIndent(req, "", "  ")
	return string(bytes)
}

type dalleResponse1 struct {
	Data []struct {
		Url string `json:"url"`
	} `json:"data"`
}
