package prompt

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"text/template"
	"time"

	"github.com/TrueBlocks/trueblocks-dalle/v6/pkg/utils"
)

// AiConfiguration holds configuration for both prompt enhancement and image generation
type AiConfiguration struct {
	// Prompt Enhancement Configuration
	EnhancementModel       string        `json:"enhancement_model"`
	EnhancementSeed        int           `json:"enhancement_seed"`
	EnhancementTemperature float64       `json:"enhancement_temperature"`
	EnhancementURL         string        `json:"enhancement_url"`
	EnhancementTimeout     time.Duration `json:"enhancement_timeout"`

	// Image Generation Configuration
	ImageModel   string        `json:"image_model"`
	ImageQuality string        `json:"image_quality"`
	ImageStyle   string        `json:"image_style"`
	ImageURL     string        `json:"image_url"`
	ImageTimeout time.Duration `json:"image_timeout"`
}

// DefaultAiConfiguration returns the default AI configuration
func DefaultAiConfiguration() AiConfiguration {
	return AiConfiguration{
		// Enhancement defaults
		EnhancementModel:       utils.GetEnvString("TB_DALLE_ENHANCEMENT_MODEL", "gpt-4"),
		EnhancementSeed:        utils.GetEnvInt("TB_DALLE_ENHANCEMENT_SEED", 1337),
		EnhancementTemperature: utils.GetEnvFloat("TB_DALLE_ENHANCEMENT_TEMPERATURE", 0.2),
		EnhancementURL:         utils.GetEnvString("TB_DALLE_ENHANCEMENT_URL", "https://api.openai.com/v1/chat/completions"),
		EnhancementTimeout:     utils.GetEnvDuration("TB_DALLE_ENHANCEMENT_TIMEOUT", 60*time.Second),

		// Image generation defaults
		ImageModel:   utils.GetEnvString("TB_DALLE_IMAGE_MODEL", "dall-e-3"),
		ImageQuality: utils.GetEnvString("TB_DALLE_IMAGE_QUALITY", "hd"),
		ImageStyle:   utils.GetEnvString("TB_DALLE_IMAGE_STYLE", "vivid"),
		ImageURL:     utils.GetEnvString("TB_DALLE_IMAGE_URL", "https://api.openai.com/v1/images/generations"),
		ImageTimeout: utils.GetEnvDuration("TB_DALLE_IMAGE_TIMEOUT", 300*time.Second),
	}
}

// Template strings and compiled templates
const promptTemplateStr = `{{.LitPrompt false}}Here's the prompt:

Draw a {{.Adverb false}} {{.Adjective false}} {{.Noun true}} with human-like
characteristics feeling {{.Emotion false}}{{.Occupation false}}.

Noun: {{.Noun false}} with human-like characteristics.
Emotion: {{.Emotion false}}.
Occupation: {{.Occupation false}}.
Action: {{.Action false}}.
Artistic style: {{.ArtStyle false 1}}.
{{if .HasLitStyle}}Literary Style: {{.LitStyle false}}.
{{end}}Use only the colors {{.Color true 1}} and {{.Color true 2}}.
{{.Viewpoint false}}.
{{.Composition false}}.
{{.BackStyle false}}.

Emphasize the emotional aspect of the image. Look deeply into and expand upon the
many connotative meanings of "{{.Noun true}}," "{{.Emotion true}}," "{{.Adjective true}}",
and "{{.Adverb true}}." Find the representation that most closely matches all the data.

Focus on the emotion, the noun, and the styles.`

const dataTemplateStr = `
Adverb:             {{.Adverb true}}
Adjective:          {{.Adjective true}}
Noun:               {{.Noun true}}
Emotion:            {{.Emotion true}}
Occupation:         {{.Occupation true}}
Action:        	    {{.Action true}}
ArtStyle 1:         {{.ArtStyle true 1}}
ArtStyle 2:         {{.ArtStyle true 2}}
{{if .HasLitStyle}}LitStyle:           {{.LitStyle false}}
{{end}}Viewpoint:          {{.Viewpoint true}}
Gaze:               {{.Gaze true}}
BackStyle:          {{.BackStyle true}}
Composition:        {{.Composition true}}
Color 1:            {{.Color false 1}}
Color 2:            {{.Color false 2}}
Color 3:            {{.Color false 3}}
------------------------------------------
Original:           {{.Original}}
Filename:           {{.Filename}}
Seed:               {{.Seed}}
Adverb (full):      {{.Adverb false}}
Adjective (full):   {{.Adjective false}}
Noun (full):        {{.Noun false}}
Emotion (full):     {{.Emotion false}}
Occupation (full):  {{.Occupation false}}
Action (full):      {{.Action false}}
ArtStyle 1 (full):  {{.ArtStyle false 1}}
ArtStyle 2 (full):  {{.ArtStyle false 2}}
{{if .HasLitStyle}}LitStyle (full):    {{.LitStyle true}}
{{end}}Viewpoint (full):   {{.Viewpoint false}}
Gaze (full):        {{.Gaze false}}
BackStyle:          {{.BackStyle false}}
Composition (full): {{.Composition false}}`

const terseTemplateStr = `{{.Adverb false}} {{.Adjective false}} {{.Noun true}} with human-like characteristics feeling {{.Emotion false}}{{.Occupation false}} in the style of {{.ArtStyle true 1}}`

const titleTemplateStr = `{{.Emotion true}} {{.Adverb true}} {{.Adjective true}} {{.Occupation true}} {{.Noun true}}`

const authorTemplateStr = `{{if .HasLitStyle}}You are an award winning author who writes in the literary
style called {{.LitStyle true}}. Take on the persona of such an author.
{{.LitStyle true}} is a genre or literary style that {{.LitStyleDescr}}.{{end}}`

const technicalTemplateStr = `Technical Specifications:
- Artistic style: {{.ArtStyle false 1}}{{if ne (.ArtStyle true 1) (.ArtStyle true 2)}} with {{.ArtStyle false 2}} influences{{end}}
- Color palette: Use {{.Color true 1}} and {{.Color true 2}}
- Composition style: {{.Composition false}}
- Background approach: {{.BackStyle false}}
- Camera/viewpoint: {{.Viewpoint false}}
- Subject gaze direction: {{.Gaze false}}

Quality Standards:
- Give the central figure distinct human-like characteristics
- Maintain emotional authenticity and depth, particularly focusing on {{.Emotion false}}
- Focus on connotative meanings and cultural associations
- Create compelling visual narrative

DO NOT PUT TEXT IN THE IMAGE.`

var (
	PromptTemplate    = template.Must(template.New("prompt").Parse(promptTemplateStr))
	DataTemplate      = template.Must(template.New("data").Parse(dataTemplateStr))
	TerseTemplate     = template.Must(template.New("terse").Parse(terseTemplateStr))
	TitleTemplate     = template.Must(template.New("title").Parse(titleTemplateStr))
	AuthorTemplate    = template.Must(template.New("author").Parse(authorTemplateStr))
	TechnicalTemplate = template.Must(template.New("technical").Parse(technicalTemplateStr))
)

// EnhancePrompt calls the OpenAI API to enhance a prompt using the given author type.
func EnhancePrompt(prompt, authorType string) (string, error) {
	if os.Getenv("TB_DALLE_NO_ENHANCE") == "1" {
		return prompt, nil
	}
	apiKey := os.Getenv("OPENAI_API_KEY")
	if apiKey == "" { // no key: skip enhancement silently
		return prompt, nil
	}
	config := DefaultAiConfiguration()
	return enhancePromptWithClient(prompt, authorType, &http.Client{}, apiKey, config, json.Marshal)
}

// enhancePromptWithClient is like EnhancePrompt but allows injecting an HTTP client, API key, config, and marshal function (for testing).
func enhancePromptWithClient(prompt, authorType string, client *http.Client, apiKey string, config AiConfiguration, marshal func(v interface{}) ([]byte, error)) (string, error) {
	// If no author context provided, skip enhancement and return original prompt
	if authorType == "" {
		return prompt, nil
	}

	payload := Request{
		Model:       config.EnhancementModel,
		Seed:        config.EnhancementSeed,
		Temperature: config.EnhancementTemperature,
	}

	payload.Messages = append(payload.Messages, Message{Role: "system", Content: authorType})
	payload.Messages = append(payload.Messages, Message{Role: "user", Content: prompt})
	payloadBytes, err := marshal(payload)
	if err != nil {
		return "", err
	}

	ctx, cancel := context.WithTimeout(context.Background(), config.EnhancementTimeout)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, "POST", config.EnhancementURL, bytes.NewBuffer(payloadBytes))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+apiKey)

	utils.DebugCurl("OPENAI CHAT (EnhancePrompt)", "POST", config.EnhancementURL, map[string]string{
		"Content-Type":  "application/json",
		"Authorization": "Bearer " + apiKey,
	}, payload)

	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		if len(bodyBytes) > 512 { // truncate to keep logs readable
			bodyBytes = bodyBytes[:512]
		}
		// Try to parse error code from JSON
		var openaiErr struct {
			Error struct {
				Code    string `json:"code"`
				Message string `json:"message"`
				Type    string `json:"type"`
			} `json:"error"`
		}
		code := "OPENAI_ERROR"
		msg := string(bodyBytes)
		if err := json.Unmarshal(bodyBytes, &openaiErr); err == nil && openaiErr.Error.Code != "" {
			code = openaiErr.Error.Code
			msg = openaiErr.Error.Message
			fmt.Printf("[DEBUG] OpenAI error code parsed: %s, message: %s\n", code, msg)
		} else {
			fmt.Printf("[DEBUG] OpenAI error code NOT parsed, fallback code: %s, raw body: %s\n", code, string(bodyBytes))
		}
		// Return a proper OpenAIAPIError so metrics and logging can extract the code
		return "", &OpenAIAPIError{
			Message:    fmt.Sprintf("enhance prompt: %s", msg),
			StatusCode: resp.StatusCode,
			RequestID:  "unused",
			Code:       code,
		}
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	type response struct {
		Choices []struct {
			Message Message `json:"message"`
		} `json:"choices"`
	}
	var r response
	if err := json.Unmarshal(body, &r); err != nil {
		return "", err
	}
	if len(r.Choices) == 0 {
		return prompt, nil
	}
	content := r.Choices[0].Message.Content
	if content == "" { // defensive
		return prompt, nil
	}
	return content, nil
}

// EnhanceLiteraryContent performs Stage 1 enhancement focused on literary and creative content
func EnhanceLiteraryContent(basePrompt, authorContext string) (string, error) {
	if os.Getenv("TB_DALLE_NO_ENHANCE") == "1" {
		return basePrompt, nil
	}
	apiKey := os.Getenv("OPENAI_API_KEY")
	if apiKey == "" {
		return basePrompt, nil
	}
	config := DefaultAiConfiguration()
	return enhanceLiteraryContentWithClient(basePrompt, authorContext, &http.Client{}, apiKey, config, json.Marshal)
}

// enhanceLiteraryContentWithClient performs Stage 1 literary enhancement with dependency injection for testing
func enhanceLiteraryContentWithClient(basePrompt, authorContext string, client *http.Client, apiKey string, config AiConfiguration, marshal func(v interface{}) ([]byte, error)) (string, error) {
	// If no author context provided, skip enhancement and return original prompt
	if authorContext == "" {
		return basePrompt, nil
	}

	systemPrompt := authorContext + "\n\nEnhance the following art generation prompt while maintaining this literary perspective. Make it more vivid and evocative while preserving all key attributes. Focus on emotional depth and narrative richness."

	payload := Request{
		Model:       config.EnhancementModel,
		Seed:        config.EnhancementSeed,
		Temperature: config.EnhancementTemperature,
	}

	payload.Messages = append(payload.Messages, Message{Role: "system", Content: systemPrompt})
	payload.Messages = append(payload.Messages, Message{Role: "user", Content: basePrompt})

	payloadBytes, err := marshal(payload)
	if err != nil {
		return "", err
	}

	ctx, cancel := context.WithTimeout(context.Background(), config.EnhancementTimeout)
	defer cancel()

	url := config.EnhancementURL
	if url == "" {
		url = "https://api.openai.com/v1/chat/completions"
	}
	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(payloadBytes))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/json")
	if apiKey != "" {
		req.Header.Set("Authorization", "Bearer "+apiKey)
	}

	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer func() {
		_ = resp.Body.Close()
	}()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	if resp.StatusCode != 200 {
		var openaiErr struct {
			Error struct {
				Message string `json:"message"`
				Code    string `json:"code"`
			} `json:"error"`
		}
		msg := string(body)
		code := "OPENAI_ERROR"
		if err := json.Unmarshal(body, &openaiErr); err == nil && openaiErr.Error.Code != "" {
			code = openaiErr.Error.Code
			msg = openaiErr.Error.Message
		}
		return "", &OpenAIAPIError{
			StatusCode: resp.StatusCode,
			Code:       code,
			Message:    msg,
		}
	}

	type response struct {
		Choices []struct {
			Message Message `json:"message"`
		} `json:"choices"`
	}
	var r response
	if err := json.Unmarshal(body, &r); err != nil {
		return "", err
	}
	if len(r.Choices) == 0 {
		return basePrompt, nil
	}
	content := r.Choices[0].Message.Content
	if content == "" {
		return basePrompt, nil
	}
	return content, nil
}
